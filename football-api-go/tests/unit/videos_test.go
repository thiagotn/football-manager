package unit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

// ── Mock store ───────────────────────────────────────────────────────────────

type mockVideoStore struct {
	match           *db.Match
	videosEnabled   bool
	groupMember     *db.GroupMember
	attendance      string
	video           *db.MatchVideo
	videoList       []db.MatchVideoWithUploader
	count           int
	users           []db.VideoUser
	player          *db.Player
	created         *db.MatchVideo
	markedUploaded  bool
	deletedVideo    bool
	updatedEnabled  *bool
	updatedPlayerID uuid.UUID
	listViewer      *uuid.UUID
	likes           map[uuid.UUID]map[uuid.UUID]bool // videoID → playerID → liked
	likers          []db.VideoLiker
	viewIncrements  int
}

func (m *mockVideoStore) ensureLikes(videoID uuid.UUID) map[uuid.UUID]bool {
	if m.likes == nil {
		m.likes = map[uuid.UUID]map[uuid.UUID]bool{}
	}
	if m.likes[videoID] == nil {
		m.likes[videoID] = map[uuid.UUID]bool{}
	}
	return m.likes[videoID]
}

func (m *mockVideoStore) IncrementVideoView(ctx context.Context, videoID uuid.UUID) error {
	m.viewIncrements++
	return nil
}
func (m *mockVideoStore) LikeMatchVideo(ctx context.Context, videoID, playerID uuid.UUID) error {
	m.ensureLikes(videoID)[playerID] = true
	return nil
}
func (m *mockVideoStore) UnlikeMatchVideo(ctx context.Context, videoID, playerID uuid.UUID) error {
	delete(m.ensureLikes(videoID), playerID)
	return nil
}
func (m *mockVideoStore) CountVideoLikes(ctx context.Context, videoID uuid.UUID) (int, error) {
	return len(m.ensureLikes(videoID)), nil
}
func (m *mockVideoStore) ListVideoLikers(ctx context.Context, videoID uuid.UUID) ([]db.VideoLiker, error) {
	return m.likers, nil
}

func (m *mockVideoStore) GetMatchByID(ctx context.Context, matchID uuid.UUID) (*db.Match, error) {
	if m.match == nil || m.match.ID != matchID {
		return nil, db.ErrNotFound
	}
	return m.match, nil
}
func (m *mockVideoStore) GetMatchByHash(ctx context.Context, hash string) (*db.Match, error) {
	if m.match == nil || m.match.Hash != hash {
		return nil, db.ErrNotFound
	}
	return m.match, nil
}
func (m *mockVideoStore) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	if m.groupMember == nil {
		return nil, db.ErrNotFound
	}
	return m.groupMember, nil
}
func (m *mockVideoStore) GroupVideosEnabled(ctx context.Context, groupID uuid.UUID) (bool, error) {
	return m.videosEnabled, nil
}
func (m *mockVideoStore) GetPlayerAttendanceStatus(ctx context.Context, matchID, playerID uuid.UUID) (string, error) {
	if m.attendance == "" {
		return "", db.ErrNotFound
	}
	return m.attendance, nil
}
func (m *mockVideoStore) CreateMatchVideo(ctx context.Context, id, matchID, uploadedBy uuid.UUID, mediaType, originalKey string) (*db.MatchVideo, error) {
	m.created = &db.MatchVideo{ID: id, MatchID: matchID, UploadedBy: uploadedBy, Status: db.VideoStatusPending, MediaType: mediaType, OriginalKey: originalKey}
	return m.created, nil
}
func (m *mockVideoStore) GetMatchVideoByID(ctx context.Context, id uuid.UUID) (*db.MatchVideo, error) {
	if m.video == nil || m.video.ID != id {
		return nil, db.ErrNotFound
	}
	return m.video, nil
}
func (m *mockVideoStore) ListMatchVideos(ctx context.Context, matchID uuid.UUID, viewer *uuid.UUID) ([]db.MatchVideoWithUploader, error) {
	m.listViewer = viewer
	return m.videoList, nil
}
func (m *mockVideoStore) CountMatchVideos(ctx context.Context, matchID uuid.UUID) (int, error) {
	return m.count, nil
}
func (m *mockVideoStore) MarkVideoUploaded(ctx context.Context, id uuid.UUID, sizeBytes int64) error {
	m.markedUploaded = true
	return nil
}
func (m *mockVideoStore) DeleteMatchVideo(ctx context.Context, id uuid.UUID) error {
	m.deletedVideo = true
	return nil
}
func (m *mockVideoStore) ListVideoUsers(ctx context.Context) ([]db.VideoUser, error) {
	return m.users, nil
}
func (m *mockVideoStore) UpdatePlayerVideosEnabled(ctx context.Context, playerID uuid.UUID, enabled bool) error {
	m.updatedEnabled = &enabled
	m.updatedPlayerID = playerID
	if m.player != nil {
		m.player.VideosEnabled = enabled
	}
	return nil
}
func (m *mockVideoStore) GetPlayerByID(ctx context.Context, id uuid.UUID) (*db.Player, error) {
	if m.player == nil || m.player.ID != id {
		return nil, db.ErrNotFound
	}
	return m.player, nil
}

// ── Fixtures ─────────────────────────────────────────────────────────────────

// fakeVideoStorage devolve um StorageService apontando para um S3 fake que
// aceita HEAD (StatObject) e DELETE. statSize < 0 → 404 no HEAD.
func fakeVideoStorage(t *testing.T, statSize int64) (*services.StorageService, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			if statSize < 0 {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Length", strconv.FormatInt(statSize, 10))
			w.Header().Set("Last-Modified", "Wed, 20 Aug 2026 12:00:00 GMT")
			w.Header().Set("ETag", `"fake"`)
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	endpoint := strings.TrimPrefix(srv.URL, "http://")
	svc, err := services.NewStorageServiceWithEndpoint(endpoint, false, "key", "secret", "rachao-media", "https://cdn.rachao.app")
	if err != nil {
		srv.Close()
		t.Fatalf("storage: %v", err)
	}
	return svc, srv
}

func videoRouter(player *db.Player, store handlers.VideoStore, storage *services.StorageService) http.Handler {
	r := chi.NewRouter()
	if player != nil {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := middleware.InjectPlayerForTest(req.Context(), player)
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
	}
	h := handlers.NewVideoHandlerWithDeps(store, storage)
	r.Post("/matches/{matchID}/videos", h.CreateUpload)
	r.Post("/matches/{matchID}/videos/{videoID}/confirm", h.ConfirmUpload)
	r.Get("/matches/public/{hash}/videos", h.ListPublicVideos)
	r.Delete("/videos/{videoID}", h.DeleteVideo)
	r.Get("/admin/video-users", h.ListVideoUsers)
	r.Patch("/admin/video-users/{userID}", h.UpdateVideoAccess)
	r.Post("/videos/{videoID}/like", h.LikeVideo)
	r.Delete("/videos/{videoID}/like", h.UnlikeVideo)
	r.Get("/videos/{videoID}/likes", h.ListVideoLikes)
	r.Post("/videos/{videoID}/view", h.RegisterView)
	return r
}

func testMatch() *db.Match {
	return &db.Match{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), GroupID: uuid.New(), Hash: "abc123hash"}
}

// ── CreateUpload ─────────────────────────────────────────────────────────────

func TestCreateVideoUpload_FlagOff(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{match: testMatch(), videosEnabled: false}
	r := videoRouter(fakePlayer(), store, storage)

	w := postJSON(r, "/matches/"+store.match.ID.String()+"/videos", `{"size_bytes":1000,"content_type":"video/mp4"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "videos not enabled")
}

func TestCreateVideoUpload_NotConfirmedForbidden(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{match: testMatch(), videosEnabled: true, attendance: "pending"}
	r := videoRouter(fakePlayer(), store, storage)

	w := postJSON(r, "/matches/"+store.match.ID.String()+"/videos", `{"size_bytes":1000,"content_type":"video/mp4"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "only confirmed players")
}

func TestCreateVideoUpload_ConfirmedGetsTicket(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{match: testMatch(), videosEnabled: true, attendance: "confirmed", count: 0}
	r := videoRouter(fakePlayer(), store, storage)

	w := postJSON(r, "/matches/"+store.match.ID.String()+"/videos", `{"size_bytes":1000,"content_type":"video/mp4"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "upload_url")
	assert.Contains(t, w.Body.String(), "X-Amz-Signature")
	assert.NotNil(t, store.created)
	assert.Contains(t, store.created.OriginalKey, "videos/original/"+store.match.ID.String())
}

func TestCreateVideoUpload_GroupAdminWithoutAttendanceAllowed(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{
		match: testMatch(), videosEnabled: true,
		groupMember: &db.GroupMember{Role: db.GroupMemberRoleAdmin},
	}
	r := videoRouter(fakePlayer(), store, storage)

	w := postJSON(r, "/matches/"+store.match.ID.String()+"/videos", `{"size_bytes":1000,"content_type":"video/mp4"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateVideoUpload_LimitReached(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{match: testMatch(), videosEnabled: true, attendance: "confirmed", count: 10}
	r := videoRouter(fakePlayer(), store, storage)

	w := postJSON(r, "/matches/"+store.match.ID.String()+"/videos", `{"size_bytes":1000,"content_type":"video/mp4"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "VIDEO_LIMIT_REACHED")
}

func TestCreateVideoUpload_TooLarge(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{match: testMatch(), videosEnabled: true, attendance: "confirmed"}
	r := videoRouter(fakePlayer(), store, storage)

	w := postJSON(r, "/matches/"+store.match.ID.String()+"/videos", `{"size_bytes":160000000,"content_type":"video/mp4"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "too large")
}

func TestCreateVideoUpload_BadContentType(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{match: testMatch(), videosEnabled: true, attendance: "confirmed"}
	r := videoRouter(fakePlayer(), store, storage)

	w := postJSON(r, "/matches/"+store.match.ID.String()+"/videos", `{"size_bytes":1000,"content_type":"image/gif"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "unsupported media type")
}

func TestCreateVideoUpload_MatchNotFound(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{videosEnabled: true}
	r := videoRouter(fakePlayer(), store, storage)

	w := postJSON(r, "/matches/"+uuid.New().String()+"/videos", `{"size_bytes":1000,"content_type":"video/mp4"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateVideoUpload_StorageNotConfigured(t *testing.T) {
	store := &mockVideoStore{match: testMatch(), videosEnabled: true, attendance: "confirmed"}
	r := videoRouter(fakePlayer(), store, nil)

	w := postJSON(r, "/matches/"+store.match.ID.String()+"/videos", `{"size_bytes":1000,"content_type":"video/mp4"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateImageUpload_AcceptedWithImageLimits(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{match: testMatch(), videosEnabled: true, attendance: "confirmed"}
	r := videoRouter(fakePlayer(), store, storage)

	w := postJSON(r, "/matches/"+store.match.ID.String()+"/videos", `{"size_bytes":1000,"content_type":"image/jpeg"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"media_type":"image"`)
	assert.NotNil(t, store.created)
	assert.Equal(t, db.MediaTypeImage, store.created.MediaType)
	assert.Contains(t, store.created.OriginalKey, ".jpg")
}

func TestCreateImageUpload_TooLargeUsesImageLimit(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{match: testMatch(), videosEnabled: true, attendance: "confirmed"}
	r := videoRouter(fakePlayer(), store, storage)

	// 30MB: aceito como vídeo, mas acima do limite de 25MB para imagem
	w := postJSON(r, "/matches/"+store.match.ID.String()+"/videos", `{"size_bytes":31457280,"content_type":"image/png"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "image too large")
}

// ── ConfirmUpload ────────────────────────────────────────────────────────────

func TestConfirmVideo_HappyPath(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 5000)
	defer srv.Close()
	player := fakePlayer()
	match := testMatch()
	video := &db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: player.ID, Status: db.VideoStatusPending, OriginalKey: "videos/original/x/y.mp4"}
	store := &mockVideoStore{match: match, video: video}
	r := videoRouter(player, store, storage)

	w := postJSON(r, "/matches/"+match.ID.String()+"/videos/"+video.ID.String()+"/confirm", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, store.markedUploaded)
	assert.Contains(t, w.Body.String(), db.VideoStatusUploaded)
}

func TestConfirmVideo_IdempotentWhenAlreadyUploaded(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 5000)
	defer srv.Close()
	player := fakePlayer()
	match := testMatch()
	video := &db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: player.ID, Status: db.VideoStatusProcessing}
	store := &mockVideoStore{match: match, video: video}
	r := videoRouter(player, store, storage)

	w := postJSON(r, "/matches/"+match.ID.String()+"/videos/"+video.ID.String()+"/confirm", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, store.markedUploaded)
	assert.Contains(t, w.Body.String(), db.VideoStatusProcessing)
}

func TestConfirmVideo_ObjectMissing(t *testing.T) {
	storage, srv := fakeVideoStorage(t, -1)
	defer srv.Close()
	player := fakePlayer()
	match := testMatch()
	video := &db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: player.ID, Status: db.VideoStatusPending, OriginalKey: "videos/original/x/y.mp4"}
	store := &mockVideoStore{match: match, video: video}
	r := videoRouter(player, store, storage)

	w := postJSON(r, "/matches/"+match.ID.String()+"/videos/"+video.ID.String()+"/confirm", `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "upload not found")
}

func TestConfirmVideo_NotYourUpload(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 5000)
	defer srv.Close()
	match := testMatch()
	video := &db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: uuid.New(), Status: db.VideoStatusPending}
	store := &mockVideoStore{match: match, video: video}
	r := videoRouter(fakePlayer(), store, storage)

	w := postJSON(r, "/matches/"+match.ID.String()+"/videos/"+video.ID.String()+"/confirm", `{}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ── ListPublicVideos ─────────────────────────────────────────────────────────

func TestListPublicVideos_FlagOffReturnsEmpty(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{match: testMatch(), videosEnabled: false}
	r := videoRouter(nil, store, storage)

	w := doRequest(r, "GET", "/matches/public/"+store.match.Hash+"/videos", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"videos_enabled":false`)
	assert.Contains(t, w.Body.String(), `"videos":[]`)
}

func TestListPublicVideos_AnonymousSeesOnlyReady(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	match := testMatch()
	ready := db.MatchVideoWithUploader{MatchVideo: db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: uuid.New(), Status: db.VideoStatusReady}, UploaderName: "A"}
	processing := db.MatchVideoWithUploader{MatchVideo: db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: uuid.New(), Status: db.VideoStatusProcessing}, UploaderName: "B"}
	store := &mockVideoStore{match: match, videosEnabled: true, videoList: []db.MatchVideoWithUploader{ready, processing}}
	r := videoRouter(nil, store, storage)

	w := doRequest(r, "GET", "/matches/public/"+match.Hash+"/videos", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), ready.ID.String())
	assert.NotContains(t, w.Body.String(), processing.ID.String())
	assert.Contains(t, w.Body.String(), `"can_upload":false`)
}

func TestListPublicVideos_UploaderSeesOwnProcessing(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	player := fakePlayer()
	match := testMatch()
	own := db.MatchVideoWithUploader{MatchVideo: db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: player.ID, Status: db.VideoStatusProcessing}, UploaderName: "Me"}
	store := &mockVideoStore{match: match, videosEnabled: true, attendance: "confirmed", videoList: []db.MatchVideoWithUploader{own}}
	r := videoRouter(player, store, storage)

	w := doRequest(r, "GET", "/matches/public/"+match.Hash+"/videos", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), own.ID.String())
	assert.Contains(t, w.Body.String(), `"can_upload":true`)
}

// ── DeleteVideo ──────────────────────────────────────────────────────────────

func TestDeleteVideo_ByUploader(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	player := fakePlayer()
	match := testMatch()
	video := &db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: player.ID, Status: db.VideoStatusReady}
	store := &mockVideoStore{match: match, video: video}
	r := videoRouter(player, store, storage)

	w := doRequest(r, "DELETE", "/videos/"+video.ID.String(), "")
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, store.deletedVideo)
}

func TestDeleteVideo_ByStrangerForbidden(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	match := testMatch()
	video := &db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: uuid.New(), Status: db.VideoStatusReady}
	store := &mockVideoStore{match: match, video: video} // sem groupMember → não é admin do grupo
	r := videoRouter(fakePlayer(), store, storage)

	w := doRequest(r, "DELETE", "/videos/"+video.ID.String(), "")
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, store.deletedVideo)
}

func TestDeleteVideo_BySuperAdmin(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	match := testMatch()
	video := &db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: uuid.New(), Status: db.VideoStatusReady}
	store := &mockVideoStore{match: match, video: video}
	r := videoRouter(fakePlayer(asAdmin()), store, storage)

	w := doRequest(r, "DELETE", "/videos/"+video.ID.String(), "")
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, store.deletedVideo)
}

func TestDeleteVideo_ByGroupAdmin(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	match := testMatch()
	video := &db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: uuid.New(), Status: db.VideoStatusReady}
	store := &mockVideoStore{match: match, video: video, groupMember: &db.GroupMember{Role: db.GroupMemberRoleAdmin}}
	r := videoRouter(fakePlayer(), store, storage)

	w := doRequest(r, "DELETE", "/videos/"+video.ID.String(), "")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ── Likes ────────────────────────────────────────────────────────────────────

func TestLikeVideo_ThenUnlike(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	player := fakePlayer()
	match := testMatch()
	video := &db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: uuid.New(), Status: db.VideoStatusReady}
	store := &mockVideoStore{match: match, video: video}
	r := videoRouter(player, store, storage)

	w := postJSON(r, "/videos/"+video.ID.String()+"/like", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"like_count":1`)
	assert.Contains(t, w.Body.String(), `"liked_by_me":true`)

	// Idempotente: curtir de novo não duplica
	w = postJSON(r, "/videos/"+video.ID.String()+"/like", "")
	assert.Contains(t, w.Body.String(), `"like_count":1`)

	w = doRequest(r, "DELETE", "/videos/"+video.ID.String()+"/like", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"like_count":0`)
	assert.Contains(t, w.Body.String(), `"liked_by_me":false`)
}

func TestLikeVideo_NotFound(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{}
	r := videoRouter(fakePlayer(), store, storage)

	w := postJSON(r, "/videos/"+uuid.New().String()+"/like", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListVideoLikes_PublicListsLikers(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	match := testMatch()
	video := &db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: uuid.New(), Status: db.VideoStatusReady}
	store := &mockVideoStore{
		match: match, video: video,
		likers: []db.VideoLiker{{ID: uuid.New(), Name: "Curtidor"}},
	}
	r := videoRouter(nil, store, storage)

	w := doRequest(r, "GET", "/videos/"+video.ID.String()+"/likes", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Curtidor")
	assert.Contains(t, w.Body.String(), `"count":1`)
}

func TestListPublicVideos_IncludesLikeAggregates(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	player := fakePlayer()
	match := testMatch()
	ready := db.MatchVideoWithUploader{
		MatchVideo: db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: uuid.New(), Status: db.VideoStatusReady},
		LikeCount:  3, LikedByMe: true,
	}
	store := &mockVideoStore{match: match, videosEnabled: true, videoList: []db.MatchVideoWithUploader{ready}}
	r := videoRouter(player, store, storage)

	w := doRequest(r, "GET", "/matches/public/"+match.Hash+"/videos", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"like_count":3`)
	assert.Contains(t, w.Body.String(), `"liked_by_me":true`)
	// viewer autenticado é repassado à listagem
	assert.NotNil(t, store.listViewer)
	assert.Equal(t, player.ID, *store.listViewer)
}

// ── Views ────────────────────────────────────────────────────────────────────

func TestRegisterView_Increments(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{}
	r := videoRouter(nil, store, storage) // público, sem auth

	w := postJSON(r, "/videos/"+uuid.New().String()+"/view", "")
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, 1, store.viewIncrements)
}

func TestRegisterView_BadID404(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{}
	r := videoRouter(nil, store, storage)

	w := postJSON(r, "/videos/not-a-uuid/view", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 0, store.viewIncrements)
}

func TestListPublicVideos_IncludesViewCount(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	match := testMatch()
	ready := db.MatchVideoWithUploader{
		MatchVideo: db.MatchVideo{ID: uuid.New(), MatchID: match.ID, UploadedBy: uuid.New(), Status: db.VideoStatusReady, ViewCount: 42},
	}
	store := &mockVideoStore{match: match, videosEnabled: true, videoList: []db.MatchVideoWithUploader{ready}}
	r := videoRouter(nil, store, storage)

	w := doRequest(r, "GET", "/matches/public/"+match.Hash+"/videos", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"view_count":42`)
}

// ── Admin toggle ─────────────────────────────────────────────────────────────

func TestAdminVideoUsers_ListAndToggle(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	target := fakePlayer()
	store := &mockVideoStore{
		player: target,
		users: []db.VideoUser{
			{ID: target.ID, Name: target.Name, WhatsApp: target.WhatsApp, VideosEnabled: false},
			{ID: uuid.New(), Name: "Other", WhatsApp: "+5511888880000", VideosEnabled: true},
		},
	}
	r := videoRouter(fakePlayer(asAdmin()), store, storage)

	w := doRequest(r, "GET", "/admin/video-users", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total_enabled":1`)

	w = doRequest(r, "PATCH", "/admin/video-users/"+target.ID.String(), `{"videos_enabled":true}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, store.updatedEnabled)
	assert.True(t, *store.updatedEnabled)
	assert.Equal(t, target.ID, store.updatedPlayerID)
	assert.Contains(t, w.Body.String(), `"videos_enabled":true`)
}

func TestAdminVideoUsers_ToggleUnknownUser404(t *testing.T) {
	storage, srv := fakeVideoStorage(t, 100)
	defer srv.Close()
	store := &mockVideoStore{}
	r := videoRouter(fakePlayer(asAdmin()), store, storage)

	w := doRequest(r, "PATCH", "/admin/video-users/"+uuid.New().String(), `{"videos_enabled":true}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
