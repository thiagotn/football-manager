package integration_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fluxo completo do claim de cadastro pendente:
// admin cria grupo → adiciona membro by-phone (placeholder) → gera claim-invite →
// jogador abre o link, verifica OTP (bypass) e completa → login novo funciona,
// flags limpas e convite de uso único.
func TestClaimFlow_EndToEnd(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin do Grupo")

	// Create group
	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token,
		map[string]any{"name": "Grupo Claim Test"})
	require.Equal(t, http.StatusCreated, r.Code, "create group failed: %v", r.Body)
	groupID, _ := r.Body["id"].(string)
	require.NotEmpty(t, groupID)
	registerGroupCleanup(t, groupID)

	// Add member by phone with a placeholder number
	placeholder := "+550000" + fmt.Sprintf("%07d", uuid.New().ID()%9999999)
	r = apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/members/by-phone", admin.Token,
		map[string]any{"whatsapp": placeholder, "name": "Jogador Pendente"})
	require.Equal(t, http.StatusCreated, r.Code, "by-phone failed: %v", r.Body)
	require.Equal(t, true, r.Body["is_new"])
	member, _ := r.Body["member"].(map[string]any)
	require.NotNil(t, member)
	playerObj, _ := member["player"].(map[string]any)
	require.NotNil(t, playerObj)
	assert.Equal(t, true, playerObj["pending_registration"],
		"player created by admin must be flagged pending_registration")
	pendingPlayerID, _ := playerObj["id"].(string)
	require.NotEmpty(t, pendingPlayerID)
	t.Cleanup(func() {
		id, err := uuid.Parse(pendingPlayerID)
		if err == nil {
			_, _ = getPool(t).Exec(t.Context(), `DELETE FROM players WHERE id=$1`, id)
		}
	})

	// Generate claim invite
	r = apiCall(t, srv, http.MethodPost,
		"/api/v2/groups/"+groupID+"/members/"+pendingPlayerID+"/claim-invite", admin.Token, nil)
	require.Equal(t, http.StatusCreated, r.Code, "claim-invite failed: %v", r.Body)
	claimToken, _ := r.Body["token"].(string)
	require.NotEmpty(t, claimToken)

	// Public claim info
	r = apiCall(t, srv, http.MethodGet, "/api/v2/claims/"+claimToken, "", nil)
	require.Equal(t, http.StatusOK, r.Code, "claim info failed: %v", r.Body)
	assert.Equal(t, "Jogador", r.Body["player_first_name"])
	assert.Equal(t, "Grupo Claim Test", r.Body["group_name"])

	// Player provides their REAL number → OTP (bypass) → complete
	realNumber := "+551198" + fmt.Sprintf("%07d", uuid.New().ID()%9999999)
	r = apiCall(t, srv, http.MethodPost, "/api/v2/claims/"+claimToken+"/send-otp", "",
		map[string]string{"whatsapp": realNumber})
	require.Equal(t, http.StatusOK, r.Code, "claim send-otp failed: %v", r.Body)

	r = apiCall(t, srv, http.MethodPost, "/api/v2/claims/"+claimToken+"/verify-otp", "",
		map[string]string{"whatsapp": realNumber, "otp_code": testOTPBypassCode})
	require.Equal(t, http.StatusOK, r.Code, "claim verify-otp failed: %v", r.Body)
	otpToken, _ := r.Body["otp_token"].(string)
	require.NotEmpty(t, otpToken)

	r = apiCall(t, srv, http.MethodPost, "/api/v2/claims/"+claimToken+"/complete", "",
		map[string]string{"whatsapp": realNumber, "otp_token": otpToken, "password": "senhaNova123"})
	require.Equal(t, http.StatusOK, r.Code, "claim complete failed: %v", r.Body)
	assert.NotEmpty(t, r.Body["access_token"])
	assert.Equal(t, false, r.Body["must_change_password"])

	// Login with the new credentials works
	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/login", "",
		map[string]string{"whatsapp": realNumber, "password": "senhaNova123"})
	require.Equal(t, http.StatusOK, r.Code, "login after claim failed: %v", r.Body)
	loginToken, _ := r.Body["access_token"].(string)

	// /auth/me shows flags cleared
	r = apiCall(t, srv, http.MethodGet, "/api/v2/auth/me", loginToken, nil)
	require.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, false, r.Body["pending_registration"])
	assert.Equal(t, false, r.Body["must_change_password"])
	assert.Equal(t, realNumber, r.Body["whatsapp"])

	// Member listing no longer flags the player as pending
	r = apiCall(t, srv, http.MethodGet, "/api/v2/groups/"+groupID+"/members", admin.Token, nil)
	require.Equal(t, http.StatusOK, r.Code)
	found := false
	for _, item := range r.List {
		m, _ := item.(map[string]any)
		p, _ := m["player"].(map[string]any)
		if p != nil && p["id"] == pendingPlayerID {
			found = true
			assert.Equal(t, false, p["pending_registration"])
		}
	}
	assert.True(t, found, "claimed player must still be a group member")

	// Claim invite is single-use
	r = apiCall(t, srv, http.MethodGet, "/api/v2/claims/"+claimToken, "", nil)
	assert.Equal(t, http.StatusForbidden, r.Code)
}

// Gerar claim-invite para jogador que não está pendente → 409 PLAYER_NOT_PENDING.
func TestClaimInvite_NotPendingPlayer(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin do Grupo")
	regular := registerAndLogin(t, srv, "Jogador Normal")

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token,
		map[string]any{"name": "Grupo Not Pending"})
	require.Equal(t, http.StatusCreated, r.Code)
	groupID, _ := r.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	// Add the already-registered player to the group
	r = apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/members/by-phone", admin.Token,
		map[string]any{"whatsapp": regular.WhatsApp})
	require.Equal(t, http.StatusCreated, r.Code, "add existing member failed: %v", r.Body)

	r = apiCall(t, srv, http.MethodPost,
		"/api/v2/groups/"+groupID+"/members/"+regular.ID+"/claim-invite", admin.Token, nil)
	assert.Equal(t, http.StatusConflict, r.Code)
	assert.Equal(t, "PLAYER_NOT_PENDING", r.Body["detail"])
}

// Número já usado por outro player é recusado no send-otp do claim → 409.
func TestClaimSendOTP_NumberTaken(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin do Grupo")
	other := registerAndLogin(t, srv, "Outro Jogador")

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token,
		map[string]any{"name": "Grupo Taken"})
	require.Equal(t, http.StatusCreated, r.Code)
	groupID, _ := r.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	placeholder := "+550000" + fmt.Sprintf("%07d", uuid.New().ID()%9999999)
	r = apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/members/by-phone", admin.Token,
		map[string]any{"whatsapp": placeholder, "name": "Pendente"})
	require.Equal(t, http.StatusCreated, r.Code)
	member, _ := r.Body["member"].(map[string]any)
	playerObj, _ := member["player"].(map[string]any)
	pendingPlayerID, _ := playerObj["id"].(string)
	t.Cleanup(func() {
		id, err := uuid.Parse(pendingPlayerID)
		if err == nil {
			_, _ = getPool(t).Exec(t.Context(), `DELETE FROM players WHERE id=$1`, id)
		}
	})

	r = apiCall(t, srv, http.MethodPost,
		"/api/v2/groups/"+groupID+"/members/"+pendingPlayerID+"/claim-invite", admin.Token, nil)
	require.Equal(t, http.StatusCreated, r.Code)
	claimToken, _ := r.Body["token"].(string)

	r = apiCall(t, srv, http.MethodPost, "/api/v2/claims/"+claimToken+"/send-otp", "",
		map[string]string{"whatsapp": other.WhatsApp})
	assert.Equal(t, http.StatusConflict, r.Code)
	assert.Equal(t, "WHATSAPP_TAKEN", r.Body["detail"])
}
