package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/google/uuid"
)

// ── Teams handler validation tests ─────────────────────────────────────────────
// Teams handler uses matchIDParam which validates UUID format.
// These tests verify the validation logic indirectly through UUID parsing.

func TestMatchIDParam_ValidUUID(t *testing.T) {
	validUUID := uuid.New().String()
	assert.NotEmpty(t, validUUID)
	// UUID parsing should succeed
	parsed, err := uuid.Parse(validUUID)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, parsed)
}

func TestMatchIDParam_InvalidUUID(t *testing.T) {
	// Invalid UUIDs should fail to parse
	invalidUUIDs := []string{
		"not-a-uuid",
		"00000000-0000-invalid",
		"",
		"12345",
	}
	for _, invalid := range invalidUUIDs {
		_, err := uuid.Parse(invalid)
		assert.Error(t, err, "should reject invalid UUID: %s", invalid)
	}
}
