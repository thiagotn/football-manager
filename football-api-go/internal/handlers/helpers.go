package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
)

// renderJSON writes a JSON response with the given status code.
func renderJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// renderError writes a JSON error response, using the code from APIError if available.
func renderError(w http.ResponseWriter, err error) {
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) {
		renderJSON(w, apiErr.Code, apiErr)
		return
	}
	if err == db.ErrNotFound {
		renderJSON(w, http.StatusNotFound, apierror.NotFound("not found"))
		return
	}
	renderJSON(w, http.StatusInternalServerError, map[string]string{"detail": "internal server error"})
}

// decodeJSON decodes the request body into dst.
// Returns an Unprocessable error if the body is invalid or contains unknown fields.
func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close() //nolint:errcheck
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apierror.Unprocessablef("invalid request body: %v", err)
	}
	return nil
}

// noContent writes a 204 No Content response.
func noContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
