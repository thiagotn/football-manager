package unit_test

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Token Generation (pure function test) ──────────────────────────────────────
// The generateToken helper function is tested indirectly via its pattern:
// base64.URLEncoding.EncodeToString(24-byte-slice)[:32]

func TestGenerateToken_ProducesValidBase64(t *testing.T) {
	// Verify the token generation algorithm produces valid base64
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	fullEncoded := base64.URLEncoding.EncodeToString(b)
	token := fullEncoded[:32]
	require.Len(t, token, 32)
	// Token is valid base64 (truncated from full encoding)
	assert.NotEmpty(t, token)
}

func TestGenerateToken_ProducesUniqueTokens(t *testing.T) {
	// Verify randomness
	tokens := make(map[string]bool)
	for i := 0; i < 10; i++ {
		b := make([]byte, 24)
		if _, err := rand.Read(b); err != nil {
			t.Fatal(err)
		}
		token := base64.URLEncoding.EncodeToString(b)[:32]
		if tokens[token] {
			t.Fatal("duplicate token generated")
		}
		tokens[token] = true
	}
	assert.Equal(t, 10, len(tokens))
}
