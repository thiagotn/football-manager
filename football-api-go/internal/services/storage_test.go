package services

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeS3 captura a última request recebida e responde com o status configurado.
type fakeS3 struct {
	method, path, contentType, cacheControl string
	body                                    []byte
	status                                  int
}

func (f *fakeS3) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.method = r.Method
		f.path = r.URL.Path
		f.contentType = r.Header.Get("Content-Type")
		f.cacheControl = r.Header.Get("Cache-Control")
		f.body, _ = io.ReadAll(r.Body)
		w.Header().Set("ETag", `"fake-etag"`)
		w.WriteHeader(f.status)
	}
}

func newFakeStorage(t *testing.T, f *fakeS3) (*StorageService, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	endpoint := strings.TrimPrefix(srv.URL, "http://")
	svc, err := NewStorageServiceWithEndpoint(endpoint, false, "key", "secret", "rachao-media", "https://cdn.rachao.app")
	if err != nil {
		srv.Close()
		t.Fatalf("NewStorageServiceWithEndpoint: %v", err)
	}
	return svc, srv
}

func TestUploadAvatar_PutsObjectAndReturnsCDNURL(t *testing.T) {
	f := &fakeS3{status: http.StatusOK}
	svc, srv := newFakeStorage(t, f)
	defer srv.Close()

	url, err := svc.UploadAvatar(context.Background(), "player1", "tok123", []byte("webp-bytes"))
	if err != nil {
		t.Fatalf("UploadAvatar: %v", err)
	}
	if url != "https://cdn.rachao.app/avatars/player1-tok123.webp" {
		t.Errorf("unexpected public URL: %s", url)
	}
	if f.method != http.MethodPut {
		t.Errorf("expected PUT, got %s", f.method)
	}
	if f.path != "/rachao-media/avatars/player1-tok123.webp" {
		t.Errorf("unexpected object path: %s", f.path)
	}
	if f.contentType != "image/webp" {
		t.Errorf("unexpected content-type: %s", f.contentType)
	}
	if !strings.Contains(f.cacheControl, "max-age=31536000") {
		t.Errorf("expected long-lived cache-control, got: %s", f.cacheControl)
	}
	// Em HTTP sem TLS o minio-go envia o corpo com aws-chunked signing;
	// basta garantir que o payload está lá.
	if !strings.Contains(string(f.body), "webp-bytes") {
		t.Errorf("body mismatch: %q", f.body)
	}
}

func TestUploadAvatar_ServerErrorPropagates(t *testing.T) {
	f := &fakeS3{status: http.StatusInternalServerError}
	svc, srv := newFakeStorage(t, f)
	defer srv.Close()

	_, err := svc.UploadAvatar(context.Background(), "player1", "tok123", []byte("x"))
	if err == nil {
		t.Fatal("expected error on HTTP 500")
	}
}

func TestDeleteAvatarByURL_RemovesObjectFromLegacyURL(t *testing.T) {
	f := &fakeS3{status: http.StatusNoContent}
	svc, srv := newFakeStorage(t, f)
	defer srv.Close()

	err := svc.DeleteAvatarByURL(context.Background(), "https://old.supabase.co/storage/v1/object/public/avatars/player1-old.webp")
	if err != nil {
		t.Fatalf("DeleteAvatarByURL: %v", err)
	}
	if f.method != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", f.method)
	}
	if f.path != "/rachao-media/avatars/player1-old.webp" {
		t.Errorf("unexpected object path: %s", f.path)
	}
}

func TestDeleteAvatarByURL_IgnoresForeignURL(t *testing.T) {
	f := &fakeS3{status: http.StatusNoContent}
	svc, srv := newFakeStorage(t, f)
	defer srv.Close()

	err := svc.DeleteAvatarByURL(context.Background(), "https://example.com/nada.webp")
	if err != nil {
		t.Fatalf("DeleteAvatarByURL: %v", err)
	}
	if f.method != "" {
		t.Errorf("expected no request to storage, got %s %s", f.method, f.path)
	}
}
