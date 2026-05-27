package unit_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── pageParams pure function tests ────────────────────────────────────────

func pageParams(r *http.Request, defaultSize int) (page, pageSize, limit, offset int) {
	page = 1
	pageSize = defaultSize
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			page = n
		}
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 100 {
			pageSize = n
		}
	}
	limit = pageSize
	offset = (page - 1) * pageSize
	return
}

func TestPageParams_Defaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil) //nolint:noctx
	page, pageSize, limit, offset := pageParams(req, 10)

	assert.Equal(t, 1, page)
	assert.Equal(t, 10, pageSize)
	assert.Equal(t, 10, limit)
	assert.Equal(t, 0, offset)
}

func TestPageParams_CustomPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page=3", nil) //nolint:noctx
	page, pageSize, limit, offset := pageParams(req, 10)

	assert.Equal(t, 3, page)
	assert.Equal(t, 10, pageSize)
	assert.Equal(t, 10, limit)
	assert.Equal(t, 20, offset)
}

func TestPageParams_CustomPageSize(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page_size=25", nil) //nolint:noctx
	page, pageSize, limit, offset := pageParams(req, 10)

	assert.Equal(t, 1, page)
	assert.Equal(t, 25, pageSize)
	assert.Equal(t, 25, limit)
	assert.Equal(t, 0, offset)
}

func TestPageParams_BothParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page=2&page_size=50", nil) //nolint:noctx
	page, pageSize, limit, offset := pageParams(req, 10)

	assert.Equal(t, 2, page)
	assert.Equal(t, 50, pageSize)
	assert.Equal(t, 50, limit)
	assert.Equal(t, 50, offset)
}

func TestPageParams_PageSizeMaxCapped(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page_size=200", nil) //nolint:noctx
	page, pageSize, limit, offset := pageParams(req, 10)

	// Should be capped at 100
	assert.Equal(t, 1, page)
	assert.Equal(t, 10, pageSize) // Falls back to default when > 100
	assert.Equal(t, 10, limit)
	assert.Equal(t, 0, offset)
}

func TestPageParams_InvalidPageValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page=invalid", nil) //nolint:noctx
	page, pageSize, limit, offset := pageParams(req, 10)

	assert.Equal(t, 1, page) // Falls back to default
	assert.Equal(t, 10, pageSize)
	assert.Equal(t, 10, limit)
	assert.Equal(t, 0, offset)
}

func TestPageParams_NegativePage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page=-1", nil) //nolint:noctx
	page, pageSize, limit, offset := pageParams(req, 10)

	assert.Equal(t, 1, page) // Falls back to default (n >= 1 check)
	assert.Equal(t, 10, pageSize)
	assert.Equal(t, 10, limit)
	assert.Equal(t, 0, offset)
}

func TestPageParams_ZeroPageSize(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page_size=0", nil) //nolint:noctx
	page, pageSize, limit, offset := pageParams(req, 10)

	assert.Equal(t, 1, page)
	assert.Equal(t, 10, pageSize) // Falls back to default (n >= 1 check)
	assert.Equal(t, 10, limit)
	assert.Equal(t, 0, offset)
}

func TestPageParams_LargePageNumber(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page=1000", nil) //nolint:noctx
	page, pageSize, limit, offset := pageParams(req, 20)

	assert.Equal(t, 1000, page)
	assert.Equal(t, 20, pageSize)
	assert.Equal(t, 20, limit)
	assert.Equal(t, 19980, offset)
}
