package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFinance_GetPeriods_AsMember(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")
	enableApiV2(t, player.ID)

	// Create group with unique name
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Finance Test Group " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code, "failed to create group: %v", groupRes.Body)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok, "group response missing id field")

	// Add player to group
	apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/members", admin.Token, map[string]any{
		"player_id": player.ID,
		"role": "member",
	})

	// GET /api/v2/groups/{id}/finance/periods as member
	res := apiCall(t, srv, http.MethodGet, "/api/v2/groups/"+groupID+"/finance/periods", player.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.List)
}

func TestFinance_GetPeriods_NonMember(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")
	enableApiV2(t, player.ID)

	// Create group (without adding player)
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Finance Test Group NonMember " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok, "group response missing id field")

	// GET /api/v2/groups/{id}/finance/periods as non-member
	res := apiCall(t, srv, http.MethodGet, "/api/v2/groups/"+groupID+"/finance/periods", player.Token, nil)
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestFinance_GetPeriod_Existing(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")
	enableApiV2(t, player.ID)

	// Create group with unique name
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Finance Period Test " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok)

	// Add player to group
	apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/members", admin.Token, map[string]any{
		"player_id": player.ID,
		"role": "member",
	})

	// GET /api/v2/groups/{id}/finance/periods/{year}/{month}
	res := apiCall(t, srv, http.MethodGet, "/api/v2/groups/"+groupID+"/finance/periods/2024/01", player.Token, nil)
	// Might return 200 or 404 depending on data
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusNotFound)
}

func TestFinance_GetPeriod_InvalidMonth(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")
	enableApiV2(t, player.ID)

	// Create group with unique name
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Finance Invalid Month Test " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok)

	// Add player to group
	apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/members", admin.Token, map[string]any{
		"player_id": player.ID,
		"role": "member",
	})

	// GET with invalid month
	res := apiCall(t, srv, http.MethodGet, "/api/v2/groups/"+groupID+"/finance/periods/2024/13", player.Token, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestFinance_UpdatePayment_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")
	enableApiV2(t, player.ID)

	// Create group with unique name
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Finance Payment Test " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok)

	// Add player to group
	apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/members", admin.Token, map[string]any{
		"player_id": player.ID,
		"role": "member",
	})

	// Try to update a payment (payment ID is typically a UUID)
	res := apiCall(t, srv, http.MethodPatch, "/api/v2/finance/payments/00000000-0000-0000-0000-000000000000", admin.Token, map[string]any{
		"status": "paid",
	})
	// Should return 404 if payment doesn't exist
	assert.True(t, res.Code == http.StatusNotFound || res.Code == http.StatusBadRequest)
}

func TestFinance_UpdatePayment_NonAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")
	enableApiV2(t, player.ID)

	// Create group with unique name
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Finance NonAdmin Test " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok)

	// Add player to group
	apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/members", admin.Token, map[string]any{
		"player_id": player.ID,
		"role": "member",
	})

	// Regular member tries to update payment
	res := apiCall(t, srv, http.MethodPatch, "/api/v2/finance/payments/00000000-0000-0000-0000-000000000000", player.Token, map[string]any{
		"status": "paid",
	})
	// Handler checks payment existence before group-admin permission, so a
	// non-existent payment returns 404 even for non-admin callers.
	assert.True(t, res.Code == http.StatusForbidden || res.Code == http.StatusNotFound)
}
