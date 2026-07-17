package unit_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

// gatherMetric busca o valor de uma métrica {job_name=<job>} no registry
// default. Retorna (valor, encontrada).
func gatherMetric(t *testing.T, name, job string) (float64, bool) {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "job_name" && l.GetValue() == job {
					if m.GetGauge() != nil {
						return m.GetGauge().GetValue(), true
					}
					return m.GetCounter().GetValue(), true
				}
			}
		}
	}
	return 0, false
}

func TestExecuteJobSuccessRecordsHeartbeat(t *testing.T) {
	before := time.Now().Unix()

	n, err := services.ExecuteJob("test_ok", time.Second, func(ctx context.Context) (int, error) {
		return 3, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, n)
	ts, found := gatherMetric(t, "scheduler_job_last_success_timestamp_seconds", "test_ok")
	require.True(t, found, "gauge do job deveria existir após sucesso")
	assert.GreaterOrEqual(t, int64(ts), before)
	_, failuresExist := gatherMetric(t, "scheduler_job_failures_total", "test_ok")
	assert.False(t, failuresExist, "sucesso não deve criar série de falhas")
}

func TestExecuteJobSuccessWithZeroResultStillRecords(t *testing.T) {
	// Heartbeat é liveness: 0 itens processados ainda conta como sucesso.
	_, err := services.ExecuteJob("test_zero", time.Second, func(ctx context.Context) (int, error) {
		return 0, nil
	})

	require.NoError(t, err)
	_, found := gatherMetric(t, "scheduler_job_last_success_timestamp_seconds", "test_zero")
	assert.True(t, found)
}

func TestExecuteJobErrorRecordsFailure(t *testing.T) {
	boom := errors.New("boom")

	_, err := services.ExecuteJob("test_fail", time.Second, func(ctx context.Context) (int, error) {
		return 0, boom
	})

	require.ErrorIs(t, err, boom)
	failures, found := gatherMetric(t, "scheduler_job_failures_total", "test_fail")
	require.True(t, found)
	assert.Equal(t, float64(1), failures)
	_, successExists := gatherMetric(t, "scheduler_job_last_success_timestamp_seconds", "test_fail")
	assert.False(t, successExists, "falha não deve registrar heartbeat de sucesso")
}

func TestExecuteJobTimeoutCancelsAndRecordsFailure(t *testing.T) {
	started := time.Now()

	_, err := services.ExecuteJob("test_timeout", 50*time.Millisecond, func(ctx context.Context) (int, error) {
		// Simula query pendurada: só retorna quando o contexto é cancelado.
		<-ctx.Done()
		return 0, ctx.Err()
	})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(started), time.Second, "timeout deve destravar o job rapidamente")
	failures, found := gatherMetric(t, "scheduler_job_failures_total", "test_timeout")
	require.True(t, found)
	assert.Equal(t, float64(1), failures)
}

func TestInitJobMetricsCreatesAllSeries(t *testing.T) {
	services.InitJobMetrics()

	for _, job := range []string{services.JobRecurrence, services.JobStatusSync, services.JobVoteReminder} {
		ts, found := gatherMetric(t, "scheduler_job_last_success_timestamp_seconds", job)
		require.True(t, found, "gauge de %s deveria existir após init", job)
		assert.Greater(t, ts, float64(0))
		failures, found := gatherMetric(t, "scheduler_job_failures_total", job)
		require.True(t, found, "counter de %s deveria existir após init", job)
		assert.Equal(t, float64(0), failures)
	}
}

func TestSchedulerMetricsExposedOnMetricsEndpoint(t *testing.T) {
	services.InitJobMetrics()

	rec := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "scheduler_job_last_success_timestamp_seconds")
	assert.Contains(t, body, "scheduler_job_failures_total")
	assert.Contains(t, body, `job_name="recurrence"`)
}
