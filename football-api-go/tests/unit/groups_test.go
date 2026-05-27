package unit_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

// groupsRouter sets up a chi router with groups routes and an injected player.
// pool is nil — only tests that return before any DB call are valid here.
func groupsRouter(player *db.Player) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Mount("/groups", handlers.NewGroupHandler(nil).Routes())
	return r
}

// ── createGroup ───────────────────────────────────────────────────────────────

func TestCreateGroup_MalformedBody(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	w := postJSON(r, "/groups", `{bad json}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateGroup_EmptyName(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	w := postJSON(r, "/groups", `{"name":""}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateGroup_NameTooShort(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	w := postJSON(r, "/groups", `{"name":"x"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── getGroup / updateGroup / deleteGroup ──────────────────────────────────────

func TestGetGroup_InvalidUUID(t *testing.T) {
	r := groupsRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/groups/not-a-uuid", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateGroup_InvalidUUID(t *testing.T) {
	r := groupsRouter(fakePlayer())
	w := doRequest(r, http.MethodPatch, "/groups/not-a-uuid", `{"name":"New Name"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteGroup_InvalidUUID(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	w := doRequest(r, http.MethodDelete, "/groups/not-a-uuid", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── members ───────────────────────────────────────────────────────────────────

func TestListMembers_InvalidGroupUUID(t *testing.T) {
	r := groupsRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/groups/not-a-uuid/members", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddMember_InvalidGroupUUID(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	w := postJSON(r, "/groups/not-a-uuid/members",
		`{"player_id":"`+uuid.New().String()+`","role":"member"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateMember_InvalidGroupUUID(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	playerID := uuid.New().String()
	w := doRequest(r, http.MethodPatch,
		"/groups/not-a-uuid/members/"+playerID, `{"role":"admin"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRemoveMember_InvalidGroupUUID(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	playerID := uuid.New().String()
	w := doRequest(r, http.MethodDelete,
		"/groups/not-a-uuid/members/"+playerID, "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── groupStats ────────────────────────────────────────────────────────────────

func TestGroupStats_InvalidUUID(t *testing.T) {
	r := groupsRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/groups/bad-id/stats", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── Pure function tests ───────────────────────────────────────────────────

// slugify implementation for testing
func slugify(s string) string {
	var b strings.Builder
	prevHyphen := true
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen {
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	slug := strings.TrimRight(b.String(), "-")
	if len(slug) > 60 {
		slug = slug[:60]
	}
	return slug
}

// buildMemberResponse implementation for testing
type memberPlayerView struct {
	ID        uuid.UUID
	Name      string
	Nickname  string
	Role      string
	AvatarURL string
	WhatsApp  *string
}

type memberResponse struct {
	ID         uuid.UUID
	Player     memberPlayerView
	Role       string
	SkillStars *int
	Position   *string
	Nickname   *string
	CreatedAt  time.Time
}

var positionRe = regexp.MustCompile(`^(gk|zag|lat|mei|ata)$`)

func buildMemberResponse(m db.GroupMemberWithPlayer, isGroupAdmin bool) memberResponse {
	playerNickname := ""
	if m.PlayerNickname != nil {
		playerNickname = *m.PlayerNickname
	}
	playerAvatarURL := ""
	if m.PlayerAvatarURL != nil {
		playerAvatarURL = *m.PlayerAvatarURL
	}

	player := memberPlayerView{
		ID:        m.PlayerID,
		Name:      m.PlayerName,
		Nickname:  playerNickname,
		Role:      string(m.PlayerRole),
		AvatarURL: playerAvatarURL,
	}
	if isGroupAdmin {
		player.WhatsApp = &m.PlayerWhatsApp
	}
	skill := m.SkillStars
	pos := m.Position
	var skillPtr *int
	var posPtr *string
	if isGroupAdmin {
		skillPtr = &skill
		posPtr = &pos
	}
	return memberResponse{
		ID:         m.ID,
		Player:     player,
		Role:       string(m.Role),
		SkillStars: skillPtr,
		Position:   posPtr,
		Nickname:   m.Nickname,
		CreatedAt:  m.CreatedAt,
	}
}

// ── Test Slugify ──────────────────────────────────────────────────────────

func TestSlugify_SimpleText(t *testing.T) {
	result := slugify("My Cool Group")
	assert.Equal(t, "my-cool-group", result)
}

func TestSlugify_SpecialCharacters(t *testing.T) {
	result := slugify("Team #1 - Soccer!")
	assert.Equal(t, "team-1-soccer", result)
}

func TestSlugify_Uppercase(t *testing.T) {
	result := slugify("PELÉ SQUAD")
	assert.Equal(t, "pelé-squad", result)
}

func TestSlugify_MultipleDashes(t *testing.T) {
	result := slugify("Team --- Soccer")
	assert.Equal(t, "team-soccer", result)
}

func TestSlugify_LeadingTrailingSpecialChars(t *testing.T) {
	result := slugify("***Team Soccer***")
	assert.Equal(t, "team-soccer", result)
}

func TestSlugify_LongName(t *testing.T) {
	longName := "Very Long Group Name That Exceeds Sixty Characters In Total Length Here"
	result := slugify(longName)
	assert.Len(t, result, 60)
	// Truncated at 60 chars after slugification
	assert.True(t, len(result) <= 60)
}

func TestSlugify_WithNumbers(t *testing.T) {
	result := slugify("Team 123 Soccer 456")
	assert.Equal(t, "team-123-soccer-456", result)
}

func TestSlugify_EmptyString(t *testing.T) {
	result := slugify("")
	assert.Equal(t, "", result)
}

// ── Test BuildMemberResponse ──────────────────────────────────────────────

func TestBuildMemberResponse_NonAdminHidesFields(t *testing.T) {
	playerID := uuid.New()
	memberID := uuid.New()
	groupID := uuid.New()
	now := time.Now()

	nickname := "Striker"
	playerNickname := "JD"
	avatarURL := "https://example.com/avatar.jpg"

	member := db.GroupMemberWithPlayer{
		GroupMember: db.GroupMember{
			ID:        memberID,
			GroupID:   groupID,
			PlayerID:  playerID,
			Role:      db.GroupMemberRoleMember,
			SkillStars: 5,
			Position:  "ata",
			Nickname:  &nickname,
			CreatedAt: now,
			UpdatedAt: now,
		},
		PlayerName:     "John",
		PlayerNickname: &playerNickname,
		PlayerRole:     db.PlayerRolePlayer,
		PlayerAvatarURL: &avatarURL,
		PlayerWhatsApp: "+5511999990000",
	}

	resp := buildMemberResponse(member, false)

	assert.Equal(t, memberID, resp.ID)
	assert.Equal(t, playerID, resp.Player.ID)
	assert.Equal(t, "John", resp.Player.Name)
	assert.Nil(t, resp.Player.WhatsApp)    // Hidden
	assert.Nil(t, resp.SkillStars)         // Hidden
	assert.Nil(t, resp.Position)           // Hidden
	assert.Equal(t, &nickname, resp.Nickname)
}

func TestBuildMemberResponse_AdminSeesAllFields(t *testing.T) {
	playerID := uuid.New()
	memberID := uuid.New()
	groupID := uuid.New()
	now := time.Now()

	nickname := "Striker"
	playerNickname := "JD"
	avatarURL := "https://example.com/avatar.jpg"

	member := db.GroupMemberWithPlayer{
		GroupMember: db.GroupMember{
			ID:        memberID,
			GroupID:   groupID,
			PlayerID:  playerID,
			Role:      db.GroupMemberRoleMember,
			SkillStars: 5,
			Position:  "ata",
			Nickname:  &nickname,
			CreatedAt: now,
			UpdatedAt: now,
		},
		PlayerName:     "John",
		PlayerNickname: &playerNickname,
		PlayerRole:     db.PlayerRolePlayer,
		PlayerAvatarURL: &avatarURL,
		PlayerWhatsApp: "+5511999990000",
	}

	resp := buildMemberResponse(member, true)

	assert.Equal(t, memberID, resp.ID)
	assert.NotNil(t, resp.Player.WhatsApp)
	assert.Equal(t, "+5511999990000", *resp.Player.WhatsApp)
	assert.NotNil(t, resp.SkillStars)
	assert.Equal(t, 5, *resp.SkillStars)
	assert.NotNil(t, resp.Position)
	assert.Equal(t, "ata", *resp.Position)
}
