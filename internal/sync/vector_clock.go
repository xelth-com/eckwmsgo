package sync

import (
	"encoding/json"
	"fmt"
)

// VectorClock tracks causality between distributed nodes
// Map format: {instance_id: version}
type VectorClock map[string]int64

// ClockRelation represents the relationship between two vector clocks
type ClockRelation int

const (
	ClockBefore      ClockRelation = iota // Local happened before remote
	ClockAfter                             // Local happened after remote
	ClockEqual                             // Simultaneous (same version)
	ClockConcurrent                        // Concurrent modifications (conflict)
)

// NewVectorClock creates a new vector clock
func NewVectorClock() VectorClock {
	return make(VectorClock)
}

// Increment increases the version for a specific instance
func (vc VectorClock) Increment(instanceID string) {
	vc[instanceID]++
}

// Get returns the version for a specific instance
func (vc VectorClock) Get(instanceID string) int64 {
	return vc[instanceID]
}

// Set sets the version for a specific instance
func (vc VectorClock) Set(instanceID string, version int64) {
	vc[instanceID] = version
}

// Merge merges another vector clock, taking the maximum for each instance
func (vc VectorClock) Merge(other VectorClock) {
	for instance, version := range other {
		if vc[instance] < version {
			vc[instance] = version
		}
	}
}

// Copy creates a deep copy of the vector clock
func (vc VectorClock) Copy() VectorClock {
	result := make(VectorClock, len(vc))
	for k, v := range vc {
		result[k] = v
	}
	return result
}

// Compare compares two vector clocks and returns their relationship
func (vc VectorClock) Compare(other VectorClock) ClockRelation {
	lessOrEqual := true
	greaterOrEqual := true

	// Get all unique instance IDs from both clocks
	allInstances := make(map[string]bool)
	for k := range vc {
		allInstances[k] = true
	}
	for k := range other {
		allInstances[k] = true
	}

	// Compare each instance
	for instance := range allInstances {
		v1 := vc[instance]
		v2 := other[instance]

		if v1 > v2 {
			lessOrEqual = false
		}
		if v1 < v2 {
			greaterOrEqual = false
		}
	}

	// Determine relationship
	if lessOrEqual && greaterOrEqual {
		return ClockEqual
	} else if lessOrEqual {
		return ClockBefore
	} else if greaterOrEqual {
		return ClockAfter
	}
	return ClockConcurrent
}

// String returns a human-readable representation
func (vc VectorClock) String() string {
	data, _ := json.Marshal(vc)
	return string(data)
}

// IsEmpty returns true if the vector clock has no entries
func (vc VectorClock) IsEmpty() bool {
	return len(vc) == 0
}

// IsConcurrentWith returns true if this clock is concurrent with another
func (vc VectorClock) IsConcurrentWith(other VectorClock) bool {
	return vc.Compare(other) == ClockConcurrent
}

// HappenedBefore returns true if this clock happened before another
func (vc VectorClock) HappenedBefore(other VectorClock) bool {
	relation := vc.Compare(other)
	return relation == ClockBefore || relation == ClockEqual
}

// HappenedAfter returns true if this clock happened after another
func (vc VectorClock) HappenedAfter(other VectorClock) bool {
	return vc.Compare(other) == ClockAfter
}

// MarshalJSON implements json.Marshaler
func (vc VectorClock) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]int64(vc))
}

// UnmarshalJSON implements json.Unmarshaler
func (vc *VectorClock) UnmarshalJSON(data []byte) error {
	var m map[string]int64
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*vc = VectorClock(m)
	return nil
}

// Validate checks if the vector clock is valid
func (vc VectorClock) Validate() error {
	for instance, version := range vc {
		if instance == "" {
			return fmt.Errorf("empty instance ID in vector clock")
		}
		if version < 0 {
			return fmt.Errorf("negative version %d for instance %s", version, instance)
		}
	}
	return nil
}
