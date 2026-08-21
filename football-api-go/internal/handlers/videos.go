package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

const (
	maxVideoOriginalBytes = 150 << 20 // 150 MB original enviado pelo browser
	maxVideosPerMatch     = 10
	videoPresignExpiry    = 15 * time.Minute
)

var allowedVideoContentTypes = map[string]bool{
	"video/mp4":       true,
	"video/quicktime": true, // iPhone (.mov, geralmente HEVC — o worker transcodifica)
	"video/webm":      true,
}

type VideoStore interface {
	GetMatchByID(ctx context.Context, matchID uuid.UUID) (*db.Match, error)
	GetMatchByHash(ctx context.Context, hash string) (*db.Match, error)
	GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
	GroupVideosEnabled(ctx context.Context, groupID uuid.UUID) (bool, error)
	GetPlayerAttendanceStatus(ctx context.Context, matchID, playerID uuid.UUID) (string, error)
	CreateMatchVideo(ctx context.Context, id, matchID, uploadedBy uuid.UUID, originalKey string) (*db.MatchVideo, error)
	GetMatchVideoByID(ctx context.Context, id uuid.UUID) (*db.MatchVideo, error)
	ListMatchVideos(ctx context.Context, matchID uuid.UUID, viewer *uuid.UUID) ([]db.MatchVideoWithUploader, error)
	CountMatchVideos(ctx context.Context, matchID uuid.UUID) (int, error)
	MarkVideoUploaded(ctx context.Context, id uuid.UUID, sizeBytes int64) error
	DeleteMatchVideo(ctx context.Context, id uuid.UUID) error
	ListVideoUsers(ctx context.Context) ([]db.VideoUser, error)
	UpdatePlayerVideosEnabled(ctx context.Context, playerID uuid.UUID, enabled bool) error
	GetPlayerByID(ctx context.Context, id uuid.UUID) (*db.Player, error)
	LikeMatchVideo(ctx context.Context, videoID, playerID uuid.UUID) error
	UnlikeMatchVideo(ctx context.Context, videoID, playerID uuid.UUID) error
	CountVideoLikes(ctx context.Context, videoID uuid.UUID) (int, error)
	ListVideoLikers(ctx context.Context, videoID uuid.UUID) ([]db.VideoLiker, error)
}

type pgVideoStore struct {
	pool *pgxpool.Pool
}

func (s *pgVideoStore) GetMatchByID(ctx context.Context, matchID uuid.UUID) (*db.Match, error) {
	return db.GetMatchByID(ctx, s.pool, matchID)
}
func (s *pgVideoStore) GetMatchByHash(ctx context.Context, hash string) (*db.Match, error) {
	return db.GetMatchByHash(ctx, s.pool, hash)
}
func (s *pgVideoStore) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	return db.GetGroupMember(ctx, s.pool, groupID, playerID)
}
func (s *pgVideoStore) GroupVideosEnabled(ctx context.Context, groupID uuid.UUID) (bool, error) {
	return db.GroupVideosEnabled(ctx, s.pool, groupID)
}
func (s *pgVideoStore) GetPlayerAttendanceStatus(ctx context.Context, matchID, playerID uuid.UUID) (string, error) {
	return db.GetPlayerAttendanceStatus(ctx, s.pool, matchID, playerID)
}
func (s *pgVideoStore) CreateMatchVideo(ctx context.Context, id, matchID, uploadedBy uuid.UUID, originalKey string) (*db.MatchVideo, error) {
	return db.CreateMatchVideo(ctx, s.pool, id, matchID, uploadedBy, originalKey)
}
func (s *pgVideoStore) GetMatchVideoByID(ctx context.Context, id uuid.UUID) (*db.MatchVideo, error) {
	return db.GetMatchVideoByID(ctx, s.pool, id)
}
func (s *pgVideoStore) ListMatchVideos(ctx context.Context, matchID uuid.UUID, viewer *uuid.UUID) ([]db.MatchVideoWithUploader, error) {
	return db.ListMatchVideos(ctx, s.pool, matchID, viewer)
}
func (s *pgVideoStore) CountMatchVideos(ctx context.Context, matchID uuid.UUID) (int, error) {
	return db.CountMatchVideos(ctx, s.pool, matchID)
}
func (s *pgVideoStore) MarkVideoUploaded(ctx context.Context, id uuid.UUID, sizeBytes int64) error {
	return db.MarkVideoUploaded(ctx, s.pool, id, sizeBytes)
}
func (s *pgVideoStore) DeleteMatchVideo(ctx context.Context, id uuid.UUID) error {
	return db.DeleteMatchVideo(ctx, s.pool, id)
}
func (s *pgVideoStore) ListVideoUsers(ctx context.Context) ([]db.VideoUser, error) {
	return db.ListVideoUsers(ctx, s.pool)
}
func (s *pgVideoStore) UpdatePlayerVideosEnabled(ctx context.Context, playerID uuid.UUID, enabled bool) error {
	return db.UpdatePlayerVideosEnabled(ctx, s.pool, playerID, enabled)
}
func (s *pgVideoStore) GetPlayerByID(ctx context.Context, id uuid.UUID) (*db.Player, error) {
	return db.GetPlayerByID(ctx, s.pool, id)
}
func (s *pgVideoStore) LikeMatchVideo(ctx context.Context, videoID, playerID uuid.UUID) error {
	return db.LikeMatchVideo(ctx, s.pool, videoID, playerID)
}
func (s *pgVideoStore) UnlikeMatchVideo(ctx context.Context, videoID, playerID uuid.UUID) error {
	return db.UnlikeMatchVideo(ctx, s.pool, videoID, playerID)
}
func (s *pgVideoStore) CountVideoLikes(ctx context.Context, videoID uuid.UUID) (int, error) {
	return db.CountVideoLikes(ctx, s.pool, videoID)
}
func (s *pgVideoStore) ListVideoLikers(ctx context.Context, videoID uuid.UUID) ([]db.VideoLiker, error) {
	return db.ListVideoLikers(ctx, s.pool, videoID)
}

type VideoHandler struct {
	Store   VideoStore
	storage *services.StorageService
}

func NewVideoHandler(pool *pgxpool.Pool, storage *services.StorageService) *VideoHandler {
	return &VideoHandler{Store: &pgVideoStore{pool: pool}, storage: storage}
}

// NewVideoHandlerWithDeps builds a handler with explicit deps (unit tests).
func NewVideoHandlerWithDeps(store VideoStore, storage *services.StorageService) *VideoHandler {
	return &VideoHandler{Store: store, storage: storage}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// canManageMatchVideos reports whether the player is super admin or a group admin.
func (h *VideoHandler) canManageMatchVideos(ctx context.Context, player *db.Player, groupID uuid.UUID) bool {
	if player.Role == db.PlayerRoleAdmin {
		return true
	}
	m, err := h.Store.GetGroupMember(ctx, groupID, player.ID)
	return err == nil && m != nil && m.Role == db.GroupMemberRoleAdmin
}

// canUploadMatchVideo: super admin, group admin, or confirmed attendance.
func (h *VideoHandler) canUploadMatchVideo(ctx context.Context, player *db.Player, match *db.Match) bool {
	if h.canManageMatchVideos(ctx, player, match.GroupID) {
		return true
	}
	status, err := h.Store.GetPlayerAttendanceStatus(ctx, match.ID, player.ID)
	return err == nil && status == "confirmed"
}

func videoOriginalKey(matchID, videoID uuid.UUID) string {
	return "videos/original/" + matchID.String() + "/" + videoID.String() + ".mp4"
}

type videoResp struct {
	ID              uuid.UUID          `json:"id"`
	MatchID         uuid.UUID          `json:"match_id"`
	Status          string             `json:"status"`
	VideoURL        *string            `json:"video_url"`
	PosterURL       *string            `json:"poster_url"`
	DurationSeconds *float64           `json:"duration_seconds"`
	CreatedAt       time.Time          `json:"created_at"`
	LikeCount       int                `json:"like_count"`
	LikedByMe       bool               `json:"liked_by_me"`
	Uploader        *videoUploaderResp `json:"uploader,omitempty"`
}

type videoUploaderResp struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Nickname  *string   `json:"nickname"`
	AvatarURL *string   `json:"avatar_url"`
}

func buildVideoResp(v *db.MatchVideo) videoResp {
	return videoResp{
		ID:              v.ID,
		MatchID:         v.MatchID,
		Status:          v.Status,
		VideoURL:        v.VideoURL,
		PosterURL:       v.PosterURL,
		DurationSeconds: v.DurationSeconds,
		CreatedAt:       v.CreatedAt,
	}
}

func buildVideoWithUploaderResp(v db.MatchVideoWithUploader) videoResp {
	resp := buildVideoResp(&v.MatchVideo)
	resp.LikeCount = v.LikeCount
	resp.LikedByMe = v.LikedByMe
	resp.Uploader = &videoUploaderResp{
		ID:        v.UploadedBy,
		Name:      v.UploaderName,
		Nickname:  v.UploaderNickname,
		AvatarURL: v.UploaderAvatarURL,
	}
	return resp
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// CreateUpload handles POST /matches/{matchID}/videos: validates permissions
// and limits, creates a pending row and returns a presigned PUT URL so the
// browser uploads straight to R2 (the API never touches the video bytes).
func (h *VideoHandler) CreateUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !h.storage.IsConfigured() {
		renderError(w, apierror.Internal("storage not configured"))
		return
	}
	player := middleware.PlayerFromCtx(ctx)

	matchID, err := matchIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}
	match, err := h.Store.GetMatchByID(ctx, matchID)
	if err != nil {
		renderError(w, err)
		return
	}

	enabled, err := h.Store.GroupVideosEnabled(ctx, match.GroupID)
	if err != nil {
		renderError(w, err)
		return
	}
	if !enabled {
		renderError(w, apierror.Forbidden("videos not enabled for this group"))
		return
	}

	if !h.canUploadMatchVideo(ctx, player, match) {
		renderError(w, apierror.Forbidden("only confirmed players can upload videos"))
		return
	}

	var req struct {
		SizeBytes   int64  `json:"size_bytes"`
		ContentType string `json:"content_type"`
	}
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	if !allowedVideoContentTypes[req.ContentType] {
		renderError(w, apierror.Unprocessable("unsupported video type. Use MP4, MOV or WebM"))
		return
	}
	if req.SizeBytes <= 0 || req.SizeBytes > maxVideoOriginalBytes {
		renderError(w, apierror.Unprocessable("video too large (max 150MB)"))
		return
	}

	count, err := h.Store.CountMatchVideos(ctx, matchID)
	if err != nil {
		renderError(w, err)
		return
	}
	if count >= maxVideosPerMatch {
		renderError(w, apierror.Forbidden("VIDEO_LIMIT_REACHED"))
		return
	}

	videoID := uuid.New()
	key := videoOriginalKey(matchID, videoID)
	uploadURL, err := h.storage.PresignedPutURL(ctx, key, videoPresignExpiry)
	if err != nil {
		renderError(w, apierror.Internal("could not create upload URL"))
		return
	}
	if _, err := h.Store.CreateMatchVideo(ctx, videoID, matchID, player.ID, key); err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusCreated, map[string]any{
		"video_id":       videoID,
		"upload_url":     uploadURL,
		"expires_at":     time.Now().UTC().Add(videoPresignExpiry),
		"max_size_bytes": int64(maxVideoOriginalBytes),
	})
}

// ConfirmUpload handles POST /matches/{matchID}/videos/{videoID}/confirm:
// verifies the object landed on R2 and hands the job to the worker queue.
func (h *VideoHandler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !h.storage.IsConfigured() {
		renderError(w, apierror.Internal("storage not configured"))
		return
	}
	player := middleware.PlayerFromCtx(ctx)

	matchID, err := matchIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}
	videoID, err := uuid.Parse(chi.URLParam(r, "videoID"))
	if err != nil {
		renderError(w, apierror.NotFound("video not found"))
		return
	}
	video, err := h.Store.GetMatchVideoByID(ctx, videoID)
	if err != nil {
		renderError(w, err)
		return
	}
	if video.MatchID != matchID {
		renderError(w, apierror.NotFound("video not found"))
		return
	}
	if video.UploadedBy != player.ID {
		renderError(w, apierror.Forbidden("not your upload"))
		return
	}
	// Idempotent: retrying the confirm after success just returns the row.
	if video.Status != db.VideoStatusPending {
		renderJSON(w, http.StatusOK, buildVideoResp(video))
		return
	}

	size, err := h.storage.StatObject(ctx, video.OriginalKey)
	if err != nil {
		renderError(w, apierror.Unprocessable("upload not found — try again"))
		return
	}
	if size > maxVideoOriginalBytes {
		_ = h.storage.DeleteObject(ctx, video.OriginalKey)
		_ = h.Store.DeleteMatchVideo(ctx, videoID)
		renderError(w, apierror.Unprocessable("video too large (max 150MB)"))
		return
	}

	if err := h.Store.MarkVideoUploaded(ctx, videoID, size); err != nil {
		renderError(w, err)
		return
	}
	video.Status = db.VideoStatusUploaded
	video.SizeBytes = &size
	renderJSON(w, http.StatusOK, buildVideoResp(video))
}

// ListPublicVideos handles GET /matches/public/{hash}/videos (OptionalAuth).
// Everyone sees ready videos; the authenticated caller also sees their own
// in-flight uploads so the UI can show processing/failed states.
func (h *VideoHandler) ListPublicVideos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hash := chi.URLParam(r, "hash")
	match, err := h.Store.GetMatchByHash(ctx, hash)
	if err != nil {
		renderError(w, err)
		return
	}
	player := middleware.PlayerFromCtx(ctx) // nil for anonymous visitors

	enabled, err := h.Store.GroupVideosEnabled(ctx, match.GroupID)
	if err != nil {
		renderError(w, err)
		return
	}
	if !enabled {
		renderJSON(w, http.StatusOK, map[string]any{
			"videos": []videoResp{}, "count": 0, "max_videos": maxVideosPerMatch,
			"can_upload": false, "videos_enabled": false,
		})
		return
	}

	var viewer *uuid.UUID
	if player != nil {
		viewer = &player.ID
	}
	all, err := h.Store.ListMatchVideos(ctx, match.ID, viewer)
	if err != nil {
		renderError(w, err)
		return
	}
	videos := make([]videoResp, 0, len(all))
	count := 0
	for _, v := range all {
		if v.Status != db.VideoStatusFailed {
			count++
		}
		isOwn := player != nil && v.UploadedBy == player.ID
		if v.Status == db.VideoStatusReady || isOwn {
			videos = append(videos, buildVideoWithUploaderResp(v))
		}
	}

	canUpload := false
	if player != nil {
		canUpload = h.canUploadMatchVideo(ctx, player, match)
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"videos":         videos,
		"count":          count,
		"max_videos":     maxVideosPerMatch,
		"can_upload":     canUpload,
		"videos_enabled": true,
	})
}

// DeleteVideo handles DELETE /videos/{videoID}: uploader, group admin or super admin.
func (h *VideoHandler) DeleteVideo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	player := middleware.PlayerFromCtx(ctx)

	videoID, err := uuid.Parse(chi.URLParam(r, "videoID"))
	if err != nil {
		renderError(w, apierror.NotFound("video not found"))
		return
	}
	video, err := h.Store.GetMatchVideoByID(ctx, videoID)
	if err != nil {
		renderError(w, err)
		return
	}
	match, err := h.Store.GetMatchByID(ctx, video.MatchID)
	if err != nil {
		renderError(w, err)
		return
	}

	if video.UploadedBy != player.ID && !h.canManageMatchVideos(ctx, player, match.GroupID) {
		renderError(w, apierror.Forbidden("cannot delete this video"))
		return
	}

	// Best-effort object cleanup (row deletion is what makes the video disappear).
	if h.storage.IsConfigured() {
		_ = h.storage.DeleteObject(ctx, video.OriginalKey)
		prefix := "videos/" + video.MatchID.String() + "/" + video.ID.String()
		_ = h.storage.DeleteObject(ctx, prefix+".mp4")
		_ = h.storage.DeleteObject(ctx, prefix+".webp")
	}
	if err := h.Store.DeleteMatchVideo(ctx, videoID); err != nil {
		renderError(w, err)
		return
	}
	noContent(w)
}

// ── Likes ────────────────────────────────────────────────────────────────────

func (h *VideoHandler) likeTarget(r *http.Request) (*db.MatchVideo, error) {
	videoID, err := uuid.Parse(chi.URLParam(r, "videoID"))
	if err != nil {
		return nil, apierror.NotFound("video not found")
	}
	return h.Store.GetMatchVideoByID(r.Context(), videoID)
}

// LikeVideo handles POST /videos/{videoID}/like (auth, idempotent).
func (h *VideoHandler) LikeVideo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	player := middleware.PlayerFromCtx(ctx)
	video, err := h.likeTarget(r)
	if err != nil {
		renderError(w, err)
		return
	}
	if err := h.Store.LikeMatchVideo(ctx, video.ID, player.ID); err != nil {
		renderError(w, err)
		return
	}
	count, err := h.Store.CountVideoLikes(ctx, video.ID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, map[string]any{"like_count": count, "liked_by_me": true})
}

// UnlikeVideo handles DELETE /videos/{videoID}/like (auth, idempotent).
func (h *VideoHandler) UnlikeVideo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	player := middleware.PlayerFromCtx(ctx)
	video, err := h.likeTarget(r)
	if err != nil {
		renderError(w, err)
		return
	}
	if err := h.Store.UnlikeMatchVideo(ctx, video.ID, player.ID); err != nil {
		renderError(w, err)
		return
	}
	count, err := h.Store.CountVideoLikes(ctx, video.ID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, map[string]any{"like_count": count, "liked_by_me": false})
}

// ListVideoLikes handles GET /videos/{videoID}/likes (public — a página do
// rachão é pública, então quem curtiu também é).
func (h *VideoHandler) ListVideoLikes(w http.ResponseWriter, r *http.Request) {
	video, err := h.likeTarget(r)
	if err != nil {
		renderError(w, err)
		return
	}
	likers, err := h.Store.ListVideoLikers(r.Context(), video.ID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, map[string]any{"likers": likers, "count": len(likers)})
}

// ── Admin (feature flag) ─────────────────────────────────────────────────────

// ListVideoUsers handles GET /admin/video-users.
func (h *VideoHandler) ListVideoUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Store.ListVideoUsers(r.Context())
	if err != nil {
		renderError(w, err)
		return
	}
	totalEnabled := 0
	for _, u := range users {
		if u.VideosEnabled {
			totalEnabled++
		}
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"users":         users,
		"total_enabled": totalEnabled,
	})
}

// UpdateVideoAccess handles PATCH /admin/video-users/{userID}.
func (h *VideoHandler) UpdateVideoAccess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		renderError(w, apierror.NotFound("user not found"))
		return
	}

	var req struct {
		VideosEnabled bool `json:"videos_enabled"`
	}
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	if _, err := h.Store.GetPlayerByID(ctx, userID); err != nil {
		renderError(w, apierror.NotFound("user not found"))
		return
	}
	if err := h.Store.UpdatePlayerVideosEnabled(ctx, userID, req.VideosEnabled); err != nil {
		renderError(w, err)
		return
	}
	updated, err := h.Store.GetPlayerByID(ctx, userID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"id":             updated.ID,
		"name":           updated.Name,
		"whatsapp":       updated.WhatsApp,
		"videos_enabled": updated.VideosEnabled,
		"created_at":     updated.CreatedAt,
	})
}
