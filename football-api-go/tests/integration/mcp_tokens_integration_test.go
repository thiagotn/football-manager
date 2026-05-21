package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMCPTokens_CreateToken(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	payload := map[string]any{
		"name": "My Token",
	}

	res := apiCall(t, srv, http.MethodPost, "/api/v2/mcp-tokens", p.Token, payload)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated)

	body := res.Body

	assert.NotEmpty(t, body["token"])
	assert.Equal(t, "My Token", body["name"])
}

func TestMCPTokens_CreateToken_InvalidPayload(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	payload := map[string]any{
		// missing name
	}

	res := apiCall(t, srv, http.MethodPost, "/api/v2/mcp-tokens", p.Token, payload)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestMCPTokens_CreateToken_RequiresAuth(t *testing.T) {
	srv := newTestServer(t)

	payload := map[string]any{
		"name": "My Token",
	}

	res := apiCall(t, srv, http.MethodPost, "/api/v2/mcp-tokens", "", payload)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestMCPTokens_ListTokens(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// Create a token first
	apiCall(t, srv, http.MethodPost, "/api/v2/mcp-tokens", p.Token, map[string]any{
		"name": "Token 1",
	})

	// List tokens
	res := apiCall(t, srv, http.MethodGet, "/api/v2/mcp-tokens", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)

	body := res.Body
	assert.Contains(t, body, "tokens")
}

func TestMCPTokens_ListTokens_RequiresAuth(t *testing.T) {
	srv := newTestServer(t)

	res := apiCall(t, srv, http.MethodGet, "/api/v2/mcp-tokens", "", nil)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestMCPTokens_RevokeToken(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// Create a token
	createRes := apiCall(t, srv, http.MethodPost, "/api/v2/mcp-tokens", p.Token, map[string]any{
		"name": "Token to Revoke",
	})

	createBody := createRes.Body
	tokenID := createBody["id"].(string)

	// Revoke it
	res := apiCall(t, srv, http.MethodDelete, "/api/v2/mcp-tokens/"+tokenID, p.Token, nil)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusNoContent)
}

func TestMCPTokens_RevokeToken_InvalidID(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	res := apiCall(t, srv, http.MethodDelete, "/api/v2/mcp-tokens/invalid-uuid", p.Token, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestMCPTokens_RevokeToken_NotFound(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// Try to revoke non-existent token
	res := apiCall(t, srv, http.MethodDelete, "/api/v2/mcp-tokens/00000000-0000-0000-0000-000000000001", p.Token, nil)
	assert.True(t, res.Code == http.StatusNotFound || res.Code == http.StatusForbidden)
}
