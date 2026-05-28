package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPTokens_CreateToken_ValidPayload(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")

	// POST /api/v2/mcp-tokens to create token
	res := apiCall(t, srv, http.MethodPost, "/api/v2/mcp-tokens", p.Token, map[string]any{
		"name": "My API Token",
	})
	assert.Equal(t, http.StatusCreated, res.Code)
	assert.Contains(t, res.Body, "token")
}

func TestMCPTokens_CreateToken_EmptyName(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")

	// POST with empty name
	res := apiCall(t, srv, http.MethodPost, "/api/v2/mcp-tokens", p.Token, map[string]any{
		"name": "",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestMCPTokens_CreateToken_MissingName(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")

	// POST without name field
	res := apiCall(t, srv, http.MethodPost, "/api/v2/mcp-tokens", p.Token, map[string]any{})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestMCPTokens_ListTokens_Empty(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")

	// GET /api/v2/mcp-tokens before creating any
	res := apiCall(t, srv, http.MethodGet, "/api/v2/mcp-tokens", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.List)
}

func TestMCPTokens_ListTokens_AfterCreate(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")

	// Create a token
	createRes := apiCall(t, srv, http.MethodPost, "/api/v2/mcp-tokens", p.Token, map[string]any{
		"name": "Test Token",
	})
	require.Equal(t, http.StatusCreated, createRes.Code)

	// List tokens
	listRes := apiCall(t, srv, http.MethodGet, "/api/v2/mcp-tokens", p.Token, nil)
	assert.Equal(t, http.StatusOK, listRes.Code)
	assert.NotNil(t, listRes.List)
	assert.True(t, len(listRes.List) > 0)
}

func TestMCPTokens_RevokeToken_Existing(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")

	// Create a token
	createRes := apiCall(t, srv, http.MethodPost, "/api/v2/mcp-tokens", p.Token, map[string]any{
		"name": "Test Token",
	})
	require.Equal(t, http.StatusCreated, createRes.Code)
	tokenID := createRes.Body["id"].(string)

	// Revoke the token
	revokeRes := apiCall(t, srv, http.MethodDelete, "/api/v2/mcp-tokens/"+tokenID, p.Token, nil)
	assert.Equal(t, http.StatusNoContent, revokeRes.Code)

	// Token should be gone from list
	listRes := apiCall(t, srv, http.MethodGet, "/api/v2/mcp-tokens", p.Token, nil)
	assert.Equal(t, http.StatusOK, listRes.Code)
	// Previously created token should no longer be in the list
	for _, item := range listRes.List {
		itemMap := item.(map[string]any)
		if id, ok := itemMap["id"]; ok {
			assert.NotEqual(t, tokenID, id)
		}
	}
}

func TestMCPTokens_RevokeToken_Nonexistent(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")

	// DELETE a non-existent token
	res := apiCall(t, srv, http.MethodDelete, "/api/v2/mcp-tokens/00000000-0000-0000-0000-000000000000", p.Token, nil)
	assert.True(t, res.Code == http.StatusNotFound || res.Code == http.StatusNoContent)
}

func TestMCPTokens_RevokeToken_OtherPlayerToken(t *testing.T) {
	srv := newTestServer(t)
	p1 := registerAndLogin(t, srv, "Player 1")
	p2 := registerAndLogin(t, srv, "Player 2")

	// Player 1 creates a token
	createRes := apiCall(t, srv, http.MethodPost, "/api/v2/mcp-tokens", p1.Token, map[string]any{
		"name": "Player 1's Token",
	})
	require.Equal(t, http.StatusCreated, createRes.Code)
	tokenID := createRes.Body["id"].(string)

	// Player 2 tries to revoke Player 1's token
	revokeRes := apiCall(t, srv, http.MethodDelete, "/api/v2/mcp-tokens/"+tokenID, p2.Token, nil)
	assert.Equal(t, http.StatusForbidden, revokeRes.Code)
}
