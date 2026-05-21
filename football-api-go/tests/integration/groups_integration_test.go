package integration_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroups_CreateAndGet(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Group Owner")
	enableApiV2(t, player.ID)

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", player.Token,
		map[string]any{"name": "Test Rachão Group"})
	require.Equal(t, http.StatusCreated, r.Code, "create group: %v", r.Body)
	groupID, _ := r.Body["id"].(string)
	require.NotEmpty(t, groupID)
	registerGroupCleanup(t, groupID)

	r = apiCall(t, srv, http.MethodGet, "/api/v2/groups/"+groupID, player.Token, nil)
	require.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, "Test Rachão Group", r.Body["name"])
}

func TestGroups_ListGroups(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "List Groups Player")
	enableApiV2(t, player.ID)

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", player.Token,
		map[string]any{"name": "My List Group"})
	require.Equal(t, http.StatusCreated, r.Code)
	groupID, _ := r.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	r = apiCall(t, srv, http.MethodGet, "/api/v2/groups", player.Token, nil)
	require.Equal(t, http.StatusOK, r.Code)
	assert.GreaterOrEqual(t, len(r.List), 1)
}

func TestGroups_UpdateGroup(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Update Group Player")
	enableApiV2(t, player.ID)

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", player.Token,
		map[string]any{"name": "Before Update"})
	require.Equal(t, http.StatusCreated, r.Code)
	groupID, _ := r.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	r = apiCall(t, srv, http.MethodPatch, "/api/v2/groups/"+groupID, player.Token,
		map[string]any{"name": "After Update"})
	require.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, "After Update", r.Body["name"])
}

func TestGroups_GetGroup_NotFound(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "NotFound Player")
	enableApiV2(t, player.ID)

	r := apiCall(t, srv, http.MethodGet,
		"/api/v2/groups/"+uuid.New().String(), player.Token, nil)
	assert.Equal(t, http.StatusNotFound, r.Code)
}

func TestGroups_ListMembers(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Members Player")
	enableApiV2(t, player.ID)

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", player.Token,
		map[string]any{"name": "Members Test Group"})
	require.Equal(t, http.StatusCreated, r.Code)
	groupID, _ := r.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	r = apiCall(t, srv, http.MethodGet, "/api/v2/groups/"+groupID+"/members", player.Token, nil)
	require.Equal(t, http.StatusOK, r.Code)
	// Creator is automatically added as group admin
	assert.GreaterOrEqual(t, len(r.List), 1)
}

func TestGroups_GetGroupStats(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Stats Player")
	enableApiV2(t, player.ID)

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", player.Token,
		map[string]any{"name": "Stats Test Group"})
	require.Equal(t, http.StatusCreated, r.Code)
	groupID, _ := r.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	r = apiCall(t, srv, http.MethodGet, "/api/v2/groups/"+groupID+"/stats", player.Token, nil)
	assert.Equal(t, http.StatusOK, r.Code)
}
