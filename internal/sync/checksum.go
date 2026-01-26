package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// ChecksumCalculator handles hash generation for entities
type ChecksumCalculator struct {
	instanceID string
}

// NewChecksumCalculator creates a new checksum calculator
func NewChecksumCalculator(instanceID string) *ChecksumCalculator {
	return &ChecksumCalculator{instanceID: instanceID}
}

// ComputeChecksum generates a deterministic hash of the entity content
func (cc *ChecksumCalculator) ComputeChecksum(entity interface{}) (string, error) {
	// 1. Convert to map to control serialization
	data, err := json.Marshal(entity)
	if err != nil {
		return "", fmt.Errorf("failed to marshal entity: %w", err)
	}

	var flatMap map[string]interface{}
	if err := json.Unmarshal(data, &flatMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	// 2. Remove ignored fields (timestamps, local IDs that shouldn't affect content sync)
	delete(flatMap, "created_at")
	delete(flatMap, "updated_at")
	delete(flatMap, "last_synced_at")
	delete(flatMap, "CreatedAt")
	delete(flatMap, "UpdatedAt")
	delete(flatMap, "LastSyncedAt")

	// 3. Sort keys to ensure deterministic output
	keys := make([]string, 0, len(flatMap))
	for k := range flatMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 4. Build canonical string
	canonical := ""
	for _, k := range keys {
		val := flatMap[k]
		// Special handling for nil or types
		if val == nil {
			canonical += fmt.Sprintf("%s:null;", k)
		} else {
			// Use generic formatting to avoid float precision issues in JSON
			canonical += fmt.Sprintf("%s:%v;", k, val)
		}
	}

	// 5. Hash it
	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:]), nil
}
