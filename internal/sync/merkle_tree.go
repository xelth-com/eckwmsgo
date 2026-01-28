package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

// MerkleTree represents a hash tree for efficient sync
type MerkleTree struct {
	db         *database.DB
	entityType string
	root       *MerkleNode
}

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Level    int               `json:"level"`    // 0=root, 1=bucket, 2=leaf
	Key      string            `json:"key"`      // bucket key or entity_id
	Hash     string            `json:"hash"`     // combined hash of children or entity hash
	Children map[string]string `json:"children"` // child key -> hash (only for non-leaf nodes)
}

// MerkleTreeRequest is sent to compare trees
type MerkleTreeRequest struct {
	EntityType string `json:"type"`
	Level      int    `json:"level"`       // which level to compare (0=root, 1=buckets)
	BucketKey  string `json:"bucket,omitempty"` // if level=1, which bucket to drill into
}

// MerkleTreeResponse returns hashes at requested level
type MerkleTreeResponse struct {
	Level    int               `json:"level"`
	Hash     string            `json:"hash"`              // hash at this level
	Children map[string]string `json:"children,omitempty"` // child hashes (if drilling down)
}

// NewMerkleTree creates a new Merkle tree for an entity type
func NewMerkleTree(db *database.DB, entityType string) *MerkleTree {
	return &MerkleTree{
		db:         db,
		entityType: entityType,
	}
}

// Build constructs the Merkle tree from entity_checksums
func (mt *MerkleTree) Build() error {
	// Get all checksums for this entity type
	var checksums []models.EntityChecksum
	if err := mt.db.DB.Where("entity_type = ?", mt.entityType).Find(&checksums).Error; err != nil {
		return fmt.Errorf("failed to fetch checksums: %w", err)
	}

	if len(checksums) == 0 {
		mt.root = &MerkleNode{
			Level:    0,
			Key:      mt.entityType,
			Hash:     "",
			Children: make(map[string]string),
		}
		return nil
	}

	// Group by bucket (first char of entity_id)
	buckets := make(map[string][]models.EntityChecksum)
	for _, cs := range checksums {
		bucket := getBucket(cs.EntityID)
		buckets[bucket] = append(buckets[bucket], cs)
	}

	// Build bucket hashes
	bucketHashes := make(map[string]string)
	for bucket, items := range buckets {
		bucketHashes[bucket] = computeBucketHash(items)
	}

	// Build root hash
	rootHash := computeRootHash(bucketHashes)

	mt.root = &MerkleNode{
		Level:    0,
		Key:      mt.entityType,
		Hash:     rootHash,
		Children: bucketHashes,
	}

	return nil
}

// GetRootHash returns the root hash of the tree
func (mt *MerkleTree) GetRootHash() string {
	if mt.root == nil {
		return ""
	}
	return mt.root.Hash
}

// GetBucketHashes returns all bucket hashes
func (mt *MerkleTree) GetBucketHashes() map[string]string {
	if mt.root == nil {
		return make(map[string]string)
	}
	return mt.root.Children
}

// GetBucketEntities returns entity hashes for a specific bucket
func (mt *MerkleTree) GetBucketEntities(bucket string) (map[string]string, error) {
	var checksums []models.EntityChecksum
	pattern := bucket + "%"
	if err := mt.db.DB.Where("entity_type = ? AND entity_id LIKE ?", mt.entityType, pattern).Find(&checksums).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, cs := range checksums {
		result[cs.EntityID] = cs.FullHash
	}
	return result, nil
}

// Compare compares this tree with a remote tree response
// Returns: missingLocal, missingRemote, different
func (mt *MerkleTree) Compare(remote *MerkleTreeResponse) (needFromRemote, needToSend []string) {
	if mt.root == nil {
		// We have nothing, need everything from remote
		for key := range remote.Children {
			needFromRemote = append(needFromRemote, key)
		}
		return
	}

	// Compare bucket by bucket
	for remoteKey, remoteHash := range remote.Children {
		localHash, exists := mt.root.Children[remoteKey]
		if !exists {
			// Remote has bucket we don't have
			needFromRemote = append(needFromRemote, remoteKey)
		} else if localHash != remoteHash {
			// Bucket exists but hash differs - need to drill down
			needFromRemote = append(needFromRemote, remoteKey)
		}
	}

	// Check for buckets we have that remote doesn't
	for localKey := range mt.root.Children {
		if _, exists := remote.Children[localKey]; !exists {
			needToSend = append(needToSend, localKey)
		}
	}

	return
}

// getBucket returns the bucket key for an entity ID
// Using first character for simple bucketing
func getBucket(entityID string) string {
	if len(entityID) == 0 {
		return "_"
	}
	return strings.ToLower(string(entityID[0]))
}

// computeBucketHash computes hash for a bucket of entities
func computeBucketHash(items []models.EntityChecksum) string {
	if len(items) == 0 {
		return ""
	}

	// Sort by entity_id for deterministic hash
	sort.Slice(items, func(i, j int) bool {
		return items[i].EntityID < items[j].EntityID
	})

	// Concatenate all hashes and compute final hash
	var combined strings.Builder
	for _, item := range items {
		combined.WriteString(item.EntityID)
		combined.WriteString(":")
		combined.WriteString(item.FullHash)
		combined.WriteString(";")
	}

	hash := sha256.Sum256([]byte(combined.String()))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for shorter hash
}

// computeRootHash computes the root hash from bucket hashes
func computeRootHash(buckets map[string]string) string {
	if len(buckets) == 0 {
		return ""
	}

	// Sort bucket keys for deterministic hash
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var combined strings.Builder
	for _, k := range keys {
		combined.WriteString(k)
		combined.WriteString(":")
		combined.WriteString(buckets[k])
		combined.WriteString(";")
	}

	hash := sha256.Sum256([]byte(combined.String()))
	return hex.EncodeToString(hash[:8])
}

// MerkleTreeSync provides sync operations using Merkle trees
type MerkleTreeSync struct {
	db *database.DB
}

// NewMerkleTreeSync creates a new Merkle tree sync service
func NewMerkleTreeSync(db *database.DB) *MerkleTreeSync {
	return &MerkleTreeSync{db: db}
}

// GetTreeState returns the current Merkle tree state for an entity type
func (mts *MerkleTreeSync) GetTreeState(entityType string) (*MerkleTreeResponse, error) {
	tree := NewMerkleTree(mts.db, entityType)
	if err := tree.Build(); err != nil {
		return nil, err
	}

	return &MerkleTreeResponse{
		Level:    0,
		Hash:     tree.GetRootHash(),
		Children: tree.GetBucketHashes(),
	}, nil
}

// GetBucketState returns entity hashes for a specific bucket
func (mts *MerkleTreeSync) GetBucketState(entityType, bucket string) (*MerkleTreeResponse, error) {
	tree := NewMerkleTree(mts.db, entityType)
	entities, err := tree.GetBucketEntities(bucket)
	if err != nil {
		return nil, err
	}

	// Compute bucket hash
	hash := ""
	if len(entities) > 0 {
		var items []models.EntityChecksum
		for id, h := range entities {
			items = append(items, models.EntityChecksum{EntityID: id, FullHash: h})
		}
		hash = computeBucketHash(items)
	}

	return &MerkleTreeResponse{
		Level:    1,
		Hash:     hash,
		Children: entities,
	}, nil
}

// FindDifferences compares local and remote trees and returns IDs that need syncing
func (mts *MerkleTreeSync) FindDifferences(entityType string, remoteRoot *MerkleTreeResponse) ([]string, error) {
	tree := NewMerkleTree(mts.db, entityType)
	if err := tree.Build(); err != nil {
		return nil, err
	}

	// Quick check - if root hashes match, nothing to sync
	if tree.GetRootHash() == remoteRoot.Hash {
		log.Printf("Merkle: %s root hashes match, no sync needed", entityType)
		return nil, nil
	}

	log.Printf("Merkle: %s root hashes differ (local=%s, remote=%s), drilling down",
		entityType, tree.GetRootHash(), remoteRoot.Hash)

	// Find differing buckets
	needFromRemote, _ := tree.Compare(remoteRoot)

	// For each differing bucket, we need all entities in that bucket from remote
	var neededIDs []string
	for _, bucket := range needFromRemote {
		// Get local entities in this bucket
		localEntities, _ := tree.GetBucketEntities(bucket)

		// Get remote entities for this bucket (from Children)
		// Note: at root level, Children contains bucket hashes, not entity hashes
		// We need to request bucket-level details from remote
		// For now, mark whole bucket as needing sync
		for id := range localEntities {
			neededIDs = append(neededIDs, id)
		}
	}

	log.Printf("Merkle: %s needs %d entities from %d differing buckets",
		entityType, len(neededIDs), len(needFromRemote))

	return neededIDs, nil
}
