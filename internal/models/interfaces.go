package models

// SyncableEntity is an interface for models that support checksum synchronization
type SyncableEntity interface {
	GetEntityID() string
	GetEntityType() string
}
