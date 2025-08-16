package domain

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// KeyID is a strongly-typed wrapper for a UUID-based key identifier.
// It ensures that key IDs are always valid UUIDs.
type KeyID struct {
	value uuid.UUID
}

// NewKeyID creates a new, random KeyID.
func NewKeyID() KeyID {
	return KeyID{value: uuid.New()}
}

// KeyIDFromString parses a string into a KeyID, returning an error if the string is not a valid UUID.
func KeyIDFromString(s string) (KeyID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return KeyID{}, fmt.Errorf("invalid key id: %w", err)
	}
	return KeyID{value: id}, nil
}

// String returns the string representation of the KeyID.
func (k KeyID) String() string {
	return k.value.String()
}

// IsZero returns true if the KeyID is the nil UUID.
func (k KeyID) IsZero() bool {
	return k.value == uuid.Nil
}

// Validate checks if the KeyID is a non-nil UUID.
func (k KeyID) Validate() error {
	if k.IsZero() {
		return fmt.Errorf("key id cannot be a nil uuid")
	}
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (k KeyID) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (k *KeyID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("key id should be a string, got %s", data)
	}
	newID, err := KeyIDFromString(s)
	if err != nil {
		return err
	}
	*k = newID
	return nil
}

const maxDescriptionLength = 255

// Description is a strongly-typed wrapper for a string, enforcing a maximum length.
type Description struct {
	value string
}

// NewDescription creates a new Description, validating its length.
func NewDescription(s string) (Description, error) {
	d := Description{value: strings.TrimSpace(s)}
	if err := d.Validate(); err != nil {
		return Description{}, err
	}
	return d, nil
}

// Validate checks if the description's length is within the allowed limit.
func (d Description) Validate() error {
	if len(d.value) > maxDescriptionLength {
		return fmt.Errorf("description exceeds maximum length of %d", maxDescriptionLength)
	}
	return nil
}

// String returns the string representation of the Description.
func (d Description) String() string {
	return d.value
}

// IsEmpty returns true if the description is empty.
func (d Description) IsEmpty() bool {
	return d.value == ""
}

const (
	maxTagKeyLength   = 63
	maxTagValueLength = 255
)

// Tag represents a key-value pair, with validation for key and value constraints.
type Tag struct {
	Key   string
	Value string
}

// NewTag creates a new Tag, validating the key and value.
func NewTag(key, value string) (Tag, error) {
	t := Tag{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)}
	if err := t.Validate(); err != nil {
		return Tag{}, err
	}
	return t, nil
}

// Validate checks the constraints for the tag's key and value.
func (t Tag) Validate() error {
	if t.Key == "" {
		return fmt.Errorf("tag key cannot be empty")
	}
	if len(t.Key) > maxTagKeyLength {
		return fmt.Errorf("tag key exceeds maximum length of %d", maxTagKeyLength)
	}
	if len(t.Value) > maxTagValueLength {
		return fmt.Errorf("tag value exceeds maximum length of %d", maxTagValueLength)
	}
	return nil
}
