package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type KeyID struct {
	value uuid.UUID
}

func NewKeyID() KeyID {
	return KeyID{value: uuid.New()}
}

func KeyIDFromString(s string) (KeyID, error) {
	if s == "" {
		return KeyID{}, fmt.Errorf("key id cannot be empty")
	}

	id, err := uuid.Parse(s)
	if err != nil {
		return KeyID{}, fmt.Errorf("invalid key id: %w", err)
	}
	return KeyID{value: id}, nil
}

func (k KeyID) String() string {
	return k.value.String()
}

func (k KeyID) IsZero() bool {
	return k.value == uuid.Nil
}

const maxDescriptionLength = 255

type Description struct {
	value string
}

func NewDescription(s string) (Description, error) {
	s = strings.TrimSpace(s)
	if len(s) > maxDescriptionLength {
		return Description{}, fmt.Errorf("description exceeds maximum length of %d", maxDescriptionLength)
	}
	return Description{value: s}, nil
}

func (d Description) String() string {
	return d.value
}

func (d Description) IsEmpty() bool {
	return d.value == ""
}

const (
	maxTagKeyLength   = 63
	maxTagValueLength = 255
)

type Tag struct {
	Key   string
	Value string
}

func NewTag(key, value string) (Tag, error) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)

	if key == "" {
		return Tag{}, fmt.Errorf("tag key cannot be empty")
	}
	if len(key) > maxTagKeyLength {
		return Tag{}, fmt.Errorf("tag key exceeds maximum length of %d", maxTagKeyLength)
	}
	if len(value) > maxTagValueLength {
		return Tag{}, fmt.Errorf("tag value exceeds maximum length of %d", maxTagValueLength)
	}
	return Tag{Key: key, Value: value}, nil
}
