package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Video status lifecycle: pending → uploaded → processing → ready | failed.
const (
	VideoStatusPending    = "pending"
	VideoStatusUploaded   = "uploaded"
	VideoStatusProcessing = "processing"
	VideoStatusReady      = "ready"
	VideoStatusFailed     = "failed"
)

// Media types accepted in the feed.
const (
	MediaTypeVideo = "video"
	MediaTypeImage = "image"
)

// MatchVideo mirrors the match_videos table.
type MatchVideo struct {
	ID              uuid.UUID `json:"id"`
	MatchID         uuid.UUID `json:"match_id"`
	UploadedBy      uuid.UUID `json:"uploaded_by"`
	Status          string    `json:"status"`
	MediaType       string    `json:"media_type"`
	OriginalKey     string    `json:"-"`
	VideoURL        *string   `json:"video_url"`
	PosterURL       *string   `json:"poster_url"`
	DurationSeconds *float64  `json:"duration_seconds"`
	SizeBytes       *int64    `json:"size_bytes"`
	Error           *string   `json:"-"`
	Attempts        int       `json:"-"`
	ViewCount       int       `json:"view_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// MatchVideoWithUploader is a video row joined with the uploader's public info
// and like aggregates for the requesting viewer.
type MatchVideoWithUploader struct {
	MatchVideo
	UploaderName      string  `json:"uploader_name"`
	UploaderNickname  *string `json:"uploader_nickname"`
	UploaderAvatarURL *string `json:"uploader_avatar_url"`
	LikeCount         int     `json:"like_count"`
	LikedByMe         bool    `json:"liked_by_me"`
}

const matchVideoCols = `
	v.id, v.match_id, v.uploaded_by, v.status, v.media_type, v.original_key,
	v.video_url, v.poster_url, v.duration_seconds, v.size_bytes,
	v.error, v.attempts, v.view_count, v.created_at, v.updated_at`

func scanMatchVideo(scanFn func(dest ...any) error) (*MatchVideo, error) {
	var v MatchVideo
	err := scanFn(
		&v.ID, &v.MatchID, &v.UploadedBy, &v.Status, &v.MediaType, &v.OriginalKey,
		&v.VideoURL, &v.PosterURL, &v.DurationSeconds, &v.SizeBytes,
		&v.Error, &v.Attempts, &v.ViewCount, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &v, nil
}

// CreateMatchVideo inserts a pending media row and returns it.
func CreateMatchVideo(ctx context.Context, pool *pgxpool.Pool, id, matchID, uploadedBy uuid.UUID, mediaType, originalKey string) (*MatchVideo, error) {
	row := pool.QueryRow(ctx, `
		INSERT INTO match_videos (id, match_id, uploaded_by, media_type, original_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+matchVideoColsBare, id, matchID, uploadedBy, mediaType, originalKey)
	return scanMatchVideo(row.Scan)
}

// matchVideoColsBare is matchVideoCols without the "v." qualifier (for INSERT ... RETURNING).
const matchVideoColsBare = `
	id, match_id, uploaded_by, status, media_type, original_key,
	video_url, poster_url, duration_seconds, size_bytes,
	error, attempts, view_count, created_at, updated_at`

// GetMatchVideoByID fetches a video row by ID.
func GetMatchVideoByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*MatchVideo, error) {
	row := pool.QueryRow(ctx, `SELECT `+matchVideoCols+` FROM match_videos v WHERE v.id = $1`, id)
	return scanMatchVideo(row.Scan)
}

// ListMatchVideos returns all videos for a match (newest first) joined with
// uploader info and like aggregates. viewer may be nil (anonymous): liked_by_me
// comes back false.
func ListMatchVideos(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID, viewer *uuid.UUID) ([]MatchVideoWithUploader, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+matchVideoCols+`,
		       p.name, p.nickname, p.avatar_url,
		       (SELECT COUNT(*) FROM match_video_likes l WHERE l.video_id = v.id)::int AS like_count,
		       ($2::uuid IS NOT NULL AND EXISTS (
		           SELECT 1 FROM match_video_likes l WHERE l.video_id = v.id AND l.player_id = $2::uuid
		       )) AS liked_by_me
		FROM match_videos v
		JOIN players p ON p.id = v.uploaded_by
		WHERE v.match_id = $1
		ORDER BY v.created_at DESC`, matchID, viewer)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]MatchVideoWithUploader, 0)
	for rows.Next() {
		var v MatchVideoWithUploader
		err := rows.Scan(
			&v.ID, &v.MatchID, &v.UploadedBy, &v.Status, &v.MediaType, &v.OriginalKey,
			&v.VideoURL, &v.PosterURL, &v.DurationSeconds, &v.SizeBytes,
			&v.Error, &v.Attempts, &v.ViewCount, &v.CreatedAt, &v.UpdatedAt,
			&v.UploaderName, &v.UploaderNickname, &v.UploaderAvatarURL,
			&v.LikeCount, &v.LikedByMe,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

// IncrementVideoView bumps the view counter of a ready media item.
// Best-effort: incrementing a missing/unready row affects 0 rows, no error.
func IncrementVideoView(ctx context.Context, pool *pgxpool.Pool, videoID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE match_videos SET view_count = view_count + 1
		WHERE id = $1 AND status = 'ready'`, videoID)
	return err
}

// ── Likes ────────────────────────────────────────────────────────────────────

// VideoLiker is a player who liked a video.
type VideoLiker struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Nickname  *string   `json:"nickname"`
	AvatarURL *string   `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

// LikeMatchVideo records a like (idempotent).
func LikeMatchVideo(ctx context.Context, pool *pgxpool.Pool, videoID, playerID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO match_video_likes (video_id, player_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, videoID, playerID)
	return err
}

// UnlikeMatchVideo removes a like (idempotent).
func UnlikeMatchVideo(ctx context.Context, pool *pgxpool.Pool, videoID, playerID uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM match_video_likes WHERE video_id = $1 AND player_id = $2`,
		videoID, playerID)
	return err
}

// CountVideoLikes returns the number of likes on a video.
func CountVideoLikes(ctx context.Context, pool *pgxpool.Pool, videoID uuid.UUID) (int, error) {
	var count int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM match_video_likes WHERE video_id = $1`, videoID).Scan(&count)
	return count, err
}

// ListVideoLikers returns who liked a video (most recent first).
func ListVideoLikers(ctx context.Context, pool *pgxpool.Pool, videoID uuid.UUID) ([]VideoLiker, error) {
	rows, err := pool.Query(ctx, `
		SELECT p.id, p.name, p.nickname, p.avatar_url, l.created_at
		FROM match_video_likes l
		JOIN players p ON p.id = l.player_id
		WHERE l.video_id = $1
		ORDER BY l.created_at DESC`, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]VideoLiker, 0)
	for rows.Next() {
		var lk VideoLiker
		if err := rows.Scan(&lk.ID, &lk.Name, &lk.Nickname, &lk.AvatarURL, &lk.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, lk)
	}
	return result, rows.Err()
}

// CountMatchVideos counts non-failed videos for a match (pending uploads hold a slot).
func CountMatchVideos(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) (int, error) {
	var count int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM match_videos WHERE match_id = $1 AND status != 'failed'`,
		matchID).Scan(&count)
	return count, err
}

// MarkVideoUploaded transitions a pending video to uploaded, recording the original size.
// Returns ErrNotFound if the row is not in pending state.
func MarkVideoUploaded(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, sizeBytes int64) error {
	tag, err := pool.Exec(ctx, `
		UPDATE match_videos SET status = 'uploaded', size_bytes = $2, updated_at = NOW()
		WHERE id = $1 AND status = 'pending'`, id, sizeBytes)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteMatchVideo removes a video row.
func DeleteMatchVideo(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM match_videos WHERE id = $1`, id)
	return err
}

// GroupVideosEnabled reports whether the videos feature is enabled for a group:
// true when at least one group admin has the videos_enabled account flag.
func GroupVideosEnabled(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) (bool, error) {
	var enabled bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM group_members gm
			JOIN players p ON p.id = gm.player_id
			WHERE gm.group_id = $1 AND gm.role = 'admin' AND p.videos_enabled
		)`, groupID).Scan(&enabled)
	return enabled, err
}

// GetPlayerAttendanceStatus returns the attendance status of a player in a match.
// Returns ErrNotFound when there is no attendance row.
func GetPlayerAttendanceStatus(ctx context.Context, pool *pgxpool.Pool, matchID, playerID uuid.UUID) (string, error) {
	var status string
	err := pool.QueryRow(ctx,
		`SELECT status FROM attendances WHERE match_id = $1 AND player_id = $2`,
		matchID, playerID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrNotFound
		}
		return "", err
	}
	return status, nil
}

// ── Admin (feature flag) ─────────────────────────────────────────────────────

// VideoUser is a row for the super-admin videos feature-flag screen.
type VideoUser struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	WhatsApp      string    `json:"whatsapp"`
	VideosEnabled bool      `json:"videos_enabled"`
	CreatedAt     time.Time `json:"created_at"`
}

// ListVideoUsers returns players that admin at least one group (flag candidates),
// excluding the super admin.
func ListVideoUsers(ctx context.Context, pool *pgxpool.Pool) ([]VideoUser, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, whatsapp, videos_enabled, created_at
		FROM players
		WHERE role != 'admin' AND active = true
		  AND EXISTS (
			SELECT 1 FROM group_members gm
			WHERE gm.player_id = players.id AND gm.role = 'admin'
		  )
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]VideoUser, 0)
	for rows.Next() {
		var u VideoUser
		if err := rows.Scan(&u.ID, &u.Name, &u.WhatsApp, &u.VideosEnabled, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdatePlayerVideosEnabled sets the videos_enabled flag for a player.
func UpdatePlayerVideosEnabled(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID, enabled bool) error {
	tag, err := pool.Exec(ctx,
		`UPDATE players SET videos_enabled = $2 WHERE id = $1`, playerID, enabled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Worker queue ─────────────────────────────────────────────────────────────

// ClaimNextVideoJob atomically claims the oldest processable video for the worker.
// Also reclaims jobs stuck in processing for >15 minutes (dead worker).
// Returns ErrNotFound when the queue is empty.
func ClaimNextVideoJob(ctx context.Context, pool *pgxpool.Pool) (*MatchVideo, error) {
	row := pool.QueryRow(ctx, `
		UPDATE match_videos SET status = 'processing', attempts = attempts + 1, updated_at = NOW()
		WHERE id = (
			SELECT id FROM match_videos
			WHERE status = 'uploaded'
			   OR (status = 'processing' AND updated_at < NOW() - INTERVAL '15 minutes')
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING `+matchVideoColsBare)
	return scanMatchVideo(row.Scan)
}

// MarkVideoReady finalizes a processed media item. videoURL and duration are
// nil for images. Returns ErrNotFound when the row no longer exists (deleted
// mid-processing) so the caller can clean up artifacts.
func MarkVideoReady(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, videoURL *string, posterURL string, durationSeconds *float64, sizeBytes int64) error {
	tag, err := pool.Exec(ctx, `
		UPDATE match_videos
		SET status = 'ready', video_url = $2, poster_url = $3,
		    duration_seconds = $4, size_bytes = $5, error = NULL, updated_at = NOW()
		WHERE id = $1`, id, videoURL, posterURL, durationSeconds, sizeBytes)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkVideoFailed marks a video as permanently failed.
func MarkVideoFailed(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, errMsg string) error {
	_, err := pool.Exec(ctx, `
		UPDATE match_videos SET status = 'failed', error = $2, updated_at = NOW()
		WHERE id = $1`, id, errMsg)
	return err
}

// RequeueVideoJob returns a processing video to the uploaded state for a later retry.
func RequeueVideoJob(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE match_videos SET status = 'uploaded', updated_at = NOW()
		WHERE id = $1 AND status = 'processing'`, id)
	return err
}

// DeleteStalePendingVideos removes pending rows older than the given age
// (abandoned uploads) and returns them so the caller can delete R2 originals.
func DeleteStalePendingVideos(ctx context.Context, pool *pgxpool.Pool, olderThan time.Duration) ([]MatchVideo, error) {
	rows, err := pool.Query(ctx, `
		DELETE FROM match_videos
		WHERE status = 'pending' AND created_at < NOW() - make_interval(secs => $1)
		RETURNING `+matchVideoColsBare, olderThan.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]MatchVideo, 0)
	for rows.Next() {
		v, err := scanMatchVideo(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *v)
	}
	return result, rows.Err()
}
