package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const storageBucket = "avatars"

// StorageService handles avatar upload/deletion via Supabase Storage HTTP API.
type StorageService struct {
	baseURL        string
	serviceRoleKey string
}

func NewStorageService(supabaseURL, serviceRoleKey string) *StorageService {
	return &StorageService{
		baseURL:        strings.TrimRight(supabaseURL, "/"),
		serviceRoleKey: serviceRoleKey,
	}
}

func (s *StorageService) IsConfigured() bool {
	return s.baseURL != "" && s.serviceRoleKey != ""
}

// ExtractStoragePath extracts the relative file path from a Supabase public URL.
// Example: ".../object/public/avatars/uuid-token.webp" → "uuid-token.webp"
func (s *StorageService) ExtractStoragePath(avatarURL string) string {
	marker := "/public/" + storageBucket + "/"
	idx := strings.Index(avatarURL, marker)
	if idx == -1 {
		return ""
	}
	return avatarURL[idx+len(marker):]
}

// UploadAvatar uploads WebP bytes to Supabase Storage and returns the public URL.
// playerID and token form the filename as "<playerID>-<token>.webp".
func (s *StorageService) UploadAvatar(ctx context.Context, playerID, token string, data []byte) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("storage not configured")
	}
	path := playerID + "-" + token + ".webp"
	uploadURL := s.baseURL + "/storage/v1/object/" + storageBucket + "/" + path

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceRoleKey)
	req.Header.Set("Content-Type", "image/webp")
	req.Header.Set("x-upsert", "true")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("storage upload failed: HTTP %d", resp.StatusCode)
	}

	return s.baseURL + "/storage/v1/object/public/" + storageBucket + "/" + path, nil
}

// DeleteAvatarByURL removes a file from Supabase Storage using its public URL.
// Best-effort: errors are silently ignored.
func (s *StorageService) DeleteAvatarByURL(ctx context.Context, avatarURL string) error {
	if !s.IsConfigured() {
		return nil
	}
	path := s.ExtractStoragePath(avatarURL)
	if path == "" {
		return nil
	}
	deleteURL := s.baseURL + "/storage/v1/object/" + storageBucket
	body, _ := json.Marshal(map[string]any{"prefixes": []string{path}})

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close() //nolint:errcheck
	return nil
}
