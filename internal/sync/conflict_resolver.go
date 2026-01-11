package sync

import (
	"encoding/json"
	"fmt"
	"time"
)

// ConflictVersion represents one version in a conflict
type ConflictVersion struct {
	Data           interface{}    `json:"data"`
	Metadata       EntityMetadata `json:"metadata"`
	Source         TruthSource    `json:"source"`
	Priority       TruthPriority  `json:"priority"`
	Timestamp      time.Time      `json:"timestamp"`
	VectorClock    VectorClock    `json:"vector_clock"`
}

// Conflict represents a sync conflict between two versions
type Conflict struct {
	ID             string             `json:"id"`
	EntityType     EntityType         `json:"entity_type"`
	EntityID       string             `json:"entity_id"`
	LocalVersion   ConflictVersion    `json:"local_version"`
	RemoteVersion  ConflictVersion    `json:"remote_version"`
	AutoResolution *ConflictResolution `json:"auto_resolution,omitempty"`
	Status         ConflictStatus     `json:"status"`
	CreatedAt      time.Time          `json:"created_at"`
	ResolvedAt     *time.Time         `json:"resolved_at,omitempty"`
}

// ConflictResolution represents the resolution of a conflict
type ConflictResolution struct {
	Strategy     ConflictResolutionStrategy `json:"strategy"`
	WinnerSource TruthSource                `json:"winner_source"`
	Reason       string                     `json:"reason"`
	ResolvedBy   *string                    `json:"resolved_by,omitempty"`
}

// ConflictResolver handles conflict resolution
type ConflictResolver struct {
	defaultStrategy ConflictResolutionStrategy
	instanceID      string
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(instanceID string, defaultStrategy ConflictResolutionStrategy) *ConflictResolver {
	if defaultStrategy == "" {
		defaultStrategy = ConflictPriorityBased
	}
	return &ConflictResolver{
		defaultStrategy: defaultStrategy,
		instanceID:      instanceID,
	}
}

// ResolveConflict automatically resolves a conflict based on rules
func (cr *ConflictResolver) ResolveConflict(conflict *Conflict) *ConflictResolution {
	local := conflict.LocalVersion
	remote := conflict.RemoteVersion

	// Step 1: Check if one is a physical action (highest priority)
	if physicalResolution := cr.checkPhysicalAction(local, remote, conflict.EntityType); physicalResolution != nil {
		return physicalResolution
	}

	// Step 2: Check vector clock causality
	clockRelation := local.VectorClock.Compare(remote.VectorClock)

	switch clockRelation {
	case ClockBefore:
		// Remote is newer (causally)
		return &ConflictResolution{
			Strategy:     ConflictLastWriteWins,
			WinnerSource: remote.Source,
			Reason:       "Remote version causally follows local (vector clock)",
		}

	case ClockAfter:
		// Local is newer (causally)
		return &ConflictResolution{
			Strategy:     ConflictLastWriteWins,
			WinnerSource: local.Source,
			Reason:       "Local version causally follows remote (vector clock)",
		}

	case ClockEqual:
		// Same causality - check hash or timestamp
		return cr.resolveEqualClocks(local, remote)

	case ClockConcurrent:
		// Concurrent modifications - use priority-based resolution
		return cr.resolveConcurrent(local, remote)
	}

	// Fallback
	return &ConflictResolution{
		Strategy:     ConflictManual,
		WinnerSource: "",
		Reason:       "Unable to automatically resolve conflict",
	}
}

// checkPhysicalAction checks if either version is from a physical action (PDA scan)
func (cr *ConflictResolver) checkPhysicalAction(local, remote ConflictVersion, entityType EntityType) *ConflictResolution {
	// Physical actions only apply to certain entity types
	if !isPhysicalActionRelevant(entityType) {
		return nil
	}

	localIsPhysical := local.Priority == PriorityPhysical
	remoteIsPhysical := remote.Priority == PriorityPhysical

	if localIsPhysical && !remoteIsPhysical {
		return &ConflictResolution{
			Strategy:     ConflictPhysicalAction,
			WinnerSource: local.Source,
			Reason:       "Local change from physical action (PDA scan) takes precedence",
		}
	}

	if remoteIsPhysical && !localIsPhysical {
		return &ConflictResolution{
			Strategy:     ConflictPhysicalAction,
			WinnerSource: remote.Source,
			Reason:       "Remote change from physical action (PDA scan) takes precedence",
		}
	}

	// Both physical - resolve by timestamp
	if localIsPhysical && remoteIsPhysical {
		if local.Timestamp.After(remote.Timestamp) {
			return &ConflictResolution{
				Strategy:     ConflictPhysicalAction,
				WinnerSource: local.Source,
				Reason:       "Local physical action is more recent",
			}
		}
		return &ConflictResolution{
			Strategy:     ConflictPhysicalAction,
			WinnerSource: remote.Source,
			Reason:       "Remote physical action is more recent",
		}
	}

	return nil
}

// resolveEqualClocks resolves conflicts when vector clocks are equal
func (cr *ConflictResolver) resolveEqualClocks(local, remote ConflictVersion) *ConflictResolution {
	// If timestamps are very close (< 5 seconds), consider it simultaneous
	timeDiff := local.Timestamp.Sub(remote.Timestamp)
	if absInt64(int64(timeDiff.Seconds())) < 5 {
		return &ConflictResolution{
			Strategy:     ConflictManual,
			WinnerSource: "",
			Reason:       "Simultaneous updates detected (within 5s window), manual resolution required",
		}
	}

	// Use timestamp to break tie
	if local.Timestamp.After(remote.Timestamp) {
		return &ConflictResolution{
			Strategy:     ConflictLastWriteWins,
			WinnerSource: local.Source,
			Reason:       fmt.Sprintf("Local timestamp (%s) is more recent than remote (%s)", local.Timestamp, remote.Timestamp),
		}
	}

	return &ConflictResolution{
		Strategy:     ConflictLastWriteWins,
		WinnerSource: remote.Source,
		Reason:       fmt.Sprintf("Remote timestamp (%s) is more recent than local (%s)", remote.Timestamp, local.Timestamp),
	}
}

// resolveConcurrent resolves concurrent modifications using priority
func (cr *ConflictResolver) resolveConcurrent(local, remote ConflictVersion) *ConflictResolution {
	// Compare priorities
	if local.Priority > remote.Priority {
		return &ConflictResolution{
			Strategy:     ConflictPriorityBased,
			WinnerSource: local.Source,
			Reason:       fmt.Sprintf("Local priority (%d) > Remote priority (%d)", local.Priority, remote.Priority),
		}
	}

	if remote.Priority > local.Priority {
		return &ConflictResolution{
			Strategy:     ConflictPriorityBased,
			WinnerSource: remote.Source,
			Reason:       fmt.Sprintf("Remote priority (%d) > Local priority (%d)", remote.Priority, local.Priority),
		}
	}

	// Priorities equal - use timestamp
	if local.Timestamp.After(remote.Timestamp) {
		return &ConflictResolution{
			Strategy:     ConflictLastWriteWins,
			WinnerSource: local.Source,
			Reason:       "Equal priorities, local timestamp is more recent",
		}
	}

	return &ConflictResolution{
		Strategy:     ConflictLastWriteWins,
		WinnerSource: remote.Source,
		Reason:       "Equal priorities, remote timestamp is more recent",
	}
}

// ManualResolve allows manual resolution of a conflict
func (cr *ConflictResolver) ManualResolve(conflict *Conflict, winnerSource TruthSource, resolvedBy string) *ConflictResolution {
	return &ConflictResolution{
		Strategy:     ConflictManual,
		WinnerSource: winnerSource,
		Reason:       "Manually resolved by user",
		ResolvedBy:   &resolvedBy,
	}
}

// ShouldAcceptRemote determines if a remote change should be accepted
func (cr *ConflictResolver) ShouldAcceptRemote(localMeta, remoteMeta *EntityMetadata) (bool, string) {
	// Compare vector clocks
	clockRelation := localMeta.VectorClock.Compare(remoteMeta.VectorClock)

	switch clockRelation {
	case ClockBefore:
		return true, "Remote is causally after local"
	case ClockAfter:
		return false, "Local is causally after remote"
	case ClockEqual:
		// Check timestamps
		if remoteMeta.UpdatedAt.After(localMeta.UpdatedAt) {
			return true, "Remote timestamp is more recent"
		}
		return false, "Local timestamp is more recent or equal"
	case ClockConcurrent:
		// Use priority
		if remoteMeta.SourcePriority > localMeta.SourcePriority {
			return true, fmt.Sprintf("Remote priority (%d) > Local priority (%d)", remoteMeta.SourcePriority, localMeta.SourcePriority)
		}
		if localMeta.SourcePriority > remoteMeta.SourcePriority {
			return false, fmt.Sprintf("Local priority (%d) > Remote priority (%d)", localMeta.SourcePriority, remoteMeta.SourcePriority)
		}
		// Equal priority - use timestamp
		if remoteMeta.UpdatedAt.After(localMeta.UpdatedAt) {
			return true, "Concurrent updates, remote timestamp is more recent"
		}
		return false, "Concurrent updates, local timestamp is more recent or equal"
	}

	return false, "Unknown clock relation"
}

// Helper functions

func isPhysicalActionRelevant(entityType EntityType) bool {
	switch entityType {
	case EntityTypeItem, EntityTypeBox, EntityTypePlace:
		return true
	default:
		return false
	}
}

func absInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// SerializeConflict converts a conflict to JSON
func SerializeConflict(conflict *Conflict) (string, error) {
	data, err := json.Marshal(conflict)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeserializeConflict converts JSON to a conflict
func DeserializeConflict(data string) (*Conflict, error) {
	var conflict Conflict
	err := json.Unmarshal([]byte(data), &conflict)
	if err != nil {
		return nil, err
	}
	return &conflict, nil
}
