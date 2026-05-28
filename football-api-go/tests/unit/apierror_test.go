package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
)

func TestBadRequest(t *testing.T) {
	err := apierror.BadRequest("invalid input")
	require.NotNil(t, err)
	assert.Equal(t, "invalid input", err.Error())
	apiErr := err.(*apierror.APIError)
	assert.Equal(t, 400, apiErr.Code)
}

func TestUnauthorized(t *testing.T) {
	err := apierror.Unauthorized()
	require.NotNil(t, err)
	assert.Equal(t, "not authenticated", err.Error())
	apiErr := err.(*apierror.APIError)
	assert.Equal(t, 401, apiErr.Code)
}

func TestTooManyRequests(t *testing.T) {
	err := apierror.TooManyRequests()
	require.NotNil(t, err)
	assert.Equal(t, "too many requests", err.Error())
	apiErr := err.(*apierror.APIError)
	assert.Equal(t, 429, apiErr.Code)
}

func TestPlanLimitExceeded(t *testing.T) {
	err := apierror.PlanLimitExceeded()
	require.NotNil(t, err)
	assert.Equal(t, "PLAN_LIMIT_EXCEEDED", err.Error())
	apiErr := err.(*apierror.APIError)
	assert.Equal(t, 403, apiErr.Code)
}

func TestInternal(t *testing.T) {
	err := apierror.Internal("database connection failed")
	require.NotNil(t, err)
	assert.Equal(t, "database connection failed", err.Error())
	apiErr := err.(*apierror.APIError)
	assert.Equal(t, 500, apiErr.Code)
}

func TestUnprocessablef(t *testing.T) {
	err := apierror.Unprocessablef("invalid format: %s", "date")
	require.NotNil(t, err)
	assert.Equal(t, "invalid format: date", err.Error())
	apiErr := err.(*apierror.APIError)
	assert.Equal(t, 422, apiErr.Code)
}
