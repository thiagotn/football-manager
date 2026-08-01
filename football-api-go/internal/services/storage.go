package services

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// avatarPrefix é o prefixo dos objetos de avatar dentro do bucket de mídia.
const avatarPrefix = "avatars/"

// legacySupabaseMarker identifica URLs públicas antigas do Supabase Storage
// ainda persistidas em players.avatar_url antes do backfill para o R2.
const legacySupabaseMarker = "/storage/v1/object/public/avatars/"

// StorageService handles avatar upload/deletion on Cloudflare R2 via the S3 API.
type StorageService struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

// NewStorageService creates a storage client for Cloudflare R2.
func NewStorageService(accountID, accessKeyID, secretAccessKey, bucket, publicBaseURL string) (*StorageService, error) {
	if accountID == "" {
		return nil, fmt.Errorf("storage: missing R2 account ID")
	}
	endpoint := accountID + ".r2.cloudflarestorage.com"
	return NewStorageServiceWithEndpoint(endpoint, true, accessKeyID, secretAccessKey, bucket, publicBaseURL)
}

// NewStorageServiceWithEndpoint targets an arbitrary S3-compatible endpoint
// (local MinIO em dev, httptest em testes).
func NewStorageServiceWithEndpoint(endpoint string, useSSL bool, accessKeyID, secretAccessKey, bucket, publicBaseURL string) (*StorageService, error) {
	if accessKeyID == "" || secretAccessKey == "" || bucket == "" || publicBaseURL == "" {
		return nil, fmt.Errorf("storage: missing configuration")
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
		Region: "auto",
	})
	if err != nil {
		return nil, fmt.Errorf("storage: %w", err)
	}
	return &StorageService{
		client:        client,
		bucket:        bucket,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}, nil
}

func (s *StorageService) IsConfigured() bool {
	return s != nil && s.client != nil
}

// ExtractStoragePath extracts the object key from a public avatar URL.
// Aceita os dois formatos: o atual (https://cdn.rachao.app/avatars/x.webp)
// e o legado do Supabase (https://<ref>.supabase.co/storage/v1/object/public/avatars/x.webp),
// que continua no banco até o backfill — nos dois casos o key é "avatars/x.webp".
func (s *StorageService) ExtractStoragePath(avatarURL string) string {
	if idx := strings.Index(avatarURL, legacySupabaseMarker); idx != -1 {
		return avatarPrefix + avatarURL[idx+len(legacySupabaseMarker):]
	}
	if rest, ok := strings.CutPrefix(avatarURL, s.publicBaseURL+"/"); ok && strings.HasPrefix(rest, avatarPrefix) {
		return rest
	}
	return ""
}

// UploadAvatar uploads WebP bytes to R2 and returns the public URL.
// playerID and token form the object key as "avatars/<playerID>-<token>.webp".
func (s *StorageService) UploadAvatar(ctx context.Context, playerID, token string, data []byte) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("storage not configured")
	}
	key := avatarPrefix + playerID + "-" + token + ".webp"
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "image/webp",
		// O nome do objeto muda a cada upload (token aleatório), então a URL é imutável.
		CacheControl: "public, max-age=31536000, immutable",
	})
	if err != nil {
		return "", fmt.Errorf("storage upload failed: %w", err)
	}
	return s.publicBaseURL + "/" + key, nil
}

// DeleteAvatarByURL removes an object from R2 using its public URL.
// Best-effort: errors are silently ignored.
func (s *StorageService) DeleteAvatarByURL(ctx context.Context, avatarURL string) error {
	if !s.IsConfigured() {
		return nil
	}
	key := s.ExtractStoragePath(avatarURL)
	if key == "" {
		return nil
	}
	_ = s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	return nil
}
