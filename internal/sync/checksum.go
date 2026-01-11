package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
)

// ChecksumCalculator handles checksum calculation for entities
type ChecksumCalculator struct {
	instanceID string
}

// NewChecksumCalculator creates a new checksum calculator
func NewChecksumCalculator(instanceID string) *ChecksumCalculator {
	return &ChecksumCalculator{
		instanceID: instanceID,
	}
}

// CalculateItemHash computes hash for an Item (leaf node)
func (cc *ChecksumCalculator) CalculateItemHash(item *models.Item) string {
	data := fmt.Sprintf(
		"%s|%s|%d|%s|%s|%v|%v",
		item.SKU,
		item.Name,
		item.Quantity,
		item.Status,
		item.UpdatedAt.Format(time.RFC3339),
		item.PlaceID,
		item.BoxID,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CalculateBoxHash computes hash for a Box with its items
func (cc *ChecksumCalculator) CalculateBoxHash(box *models.Box, itemHashes []string) string {
	// Sort hashes for deterministic result
	sort.Strings(itemHashes)

	// Content hash (box's own data)
	contentData := fmt.Sprintf(
		"%s|%s|%t|%s",
		box.Barcode,
		box.Name,
		box.IsActive,
		box.UpdatedAt.Format(time.RFC3339),
	)
	contentHash := sha256.Sum256([]byte(contentData))

	// Children hash (items in box)
	childrenData := strings.Join(itemHashes, "|")
	childrenHash := sha256.Sum256([]byte(childrenData))

	// Full hash (content + children)
	fullData := fmt.Sprintf("%x|%x", contentHash, childrenHash)
	fullHash := sha256.Sum256([]byte(fullData))

	return hex.EncodeToString(fullHash[:])
}

// CalculatePlaceHash computes hash for a Place with its boxes and items
func (cc *ChecksumCalculator) CalculatePlaceHash(place *models.Place, boxHashes, itemHashes []string) string {
	// Sort all hashes for deterministic result
	sort.Strings(boxHashes)
	sort.Strings(itemHashes)

	// Content hash (place's own data)
	contentData := fmt.Sprintf(
		"%s|%s|%t|%s",
		place.Barcode,
		place.Position,
		place.IsOccupied,
		place.UpdatedAt.Format(time.RFC3339),
	)
	contentHash := sha256.Sum256([]byte(contentData))

	// Children hash (boxes + items)
	allChildren := append(boxHashes, itemHashes...)
	sort.Strings(allChildren)
	childrenData := strings.Join(allChildren, "|")
	childrenHash := sha256.Sum256([]byte(childrenData))

	// Full hash
	fullData := fmt.Sprintf("%x|%x", contentHash, childrenHash)
	fullHash := sha256.Sum256([]byte(fullData))

	return hex.EncodeToString(fullHash[:])
}

// CalculateRackHash computes hash for a Rack with its places
func (cc *ChecksumCalculator) CalculateRackHash(rack *models.WarehouseRack, placeHashes []string) string {
	sort.Strings(placeHashes)

	// Content hash
	contentData := fmt.Sprintf(
		"%s|%d|%d|%s",
		rack.Name,
		rack.Level,
		rack.Position,
		rack.UpdatedAt.Format(time.RFC3339),
	)
	contentHash := sha256.Sum256([]byte(contentData))

	// Children hash
	childrenData := strings.Join(placeHashes, "|")
	childrenHash := sha256.Sum256([]byte(childrenData))

	// Full hash
	fullData := fmt.Sprintf("%x|%x", contentHash, childrenHash)
	fullHash := sha256.Sum256([]byte(fullData))

	return hex.EncodeToString(fullHash[:])
}

// CalculateWarehouseHash computes hash for a Warehouse with its racks
func (cc *ChecksumCalculator) CalculateWarehouseHash(warehouse *models.Warehouse, rackHashes []string) string {
	sort.Strings(rackHashes)

	// Content hash
	contentData := fmt.Sprintf(
		"%s|%s|%s",
		warehouse.Name,
		warehouse.Location,
		warehouse.UpdatedAt.Format(time.RFC3339),
	)
	contentHash := sha256.Sum256([]byte(contentData))

	// Children hash
	childrenData := strings.Join(rackHashes, "|")
	childrenHash := sha256.Sum256([]byte(childrenData))

	// Full hash
	fullData := fmt.Sprintf("%x|%x", contentHash, childrenHash)
	fullHash := sha256.Sum256([]byte(fullData))

	return hex.EncodeToString(fullHash[:])
}

// CalculateOrderHash computes hash for an Order
func (cc *ChecksumCalculator) CalculateOrderHash(order *models.Order) string {
	data := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%s",
		order.OrderNumber,
		order.OrderType,
		order.Status,
		order.CustomerName,
		order.ProductSKU,
		order.UpdatedAt.Format(time.RFC3339),
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ComputeContentHash computes just the content hash (not including children)
func (cc *ChecksumCalculator) ComputeContentHash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ComputeChildrenHash computes hash of child hashes
func (cc *ChecksumCalculator) ComputeChildrenHash(childHashes []string) string {
	if len(childHashes) == 0 {
		return ""
	}
	sort.Strings(childHashes)
	data := strings.Join(childHashes, "|")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ComputeFullHash computes combined hash from content and children hashes
func (cc *ChecksumCalculator) ComputeFullHash(contentHash, childrenHash string) string {
	var data string
	if childrenHash == "" {
		data = contentHash
	} else {
		data = fmt.Sprintf("%s|%s", contentHash, childrenHash)
	}
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CompareHashes compares two hashes and returns true if they match
func (cc *ChecksumCalculator) CompareHashes(hash1, hash2 string) bool {
	return hash1 == hash2
}

// HashString is a utility function to hash any string
func HashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

// HashBytes is a utility function to hash any byte slice
func HashBytes(b []byte) string {
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}
