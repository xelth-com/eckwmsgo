package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// OdooString is a custom string type that handles Odoo's dynamic typing.
// Odoo returns `false` (boolean) for empty text fields instead of an empty string.
// This type implements json.Unmarshaler to handle both string and bool(false).
type OdooString string

// UnmarshalJSON handles dynamic typing from Odoo
func (os *OdooString) UnmarshalJSON(data []byte) error {
	// 1. Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*os = OdooString(s)
		return nil
	}

	// 2. Try boolean (Odoo returns false for empty strings)
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		if !b {
			*os = ""
			return nil
		}
		// If true, it's weird for a string field, but let's treat as "true" string
		*os = "true"
		return nil
	}

	return errors.New("OdooString: cannot unmarshal value into string")
}

// Value implements driver.Valuer interface for database storage
func (os OdooString) Value() (driver.Value, error) {
	return string(os), nil
}

// Scan implements sql.Scanner interface for database retrieval
func (os *OdooString) Scan(value interface{}) error {
	if value == nil {
		*os = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*os = OdooString(v)
	case []byte:
		*os = OdooString(string(v))
	default:
		return fmt.Errorf("failed to scan OdooString: %v", value)
	}
	return nil
}

// String returns native string value
func (os OdooString) String() string {
	return string(os)
}
