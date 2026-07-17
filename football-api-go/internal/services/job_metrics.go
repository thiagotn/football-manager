package services

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Heartbeat Prometheus dos jobs do scheduler (PRD 049 §4).
// Espelha football-api/app/core/job_metrics.py — mesmos nomes de métrica e de
// job, para que as queries e alertas escritos para a v1 funcionem sem adaptação.
// Diferença deliberada: sem o label `pid` (a v1 roda 2 workers uvicorn; aqui o
// processo é único e as queries usam max()/sum(), que seguem válidas).

const (
	JobRecurrence   = "recurrence"
	JobStatusSync   = "status_sync"
	JobVoteReminder = "vote_reminder"
)

var allJobs = []string{JobRecurrence, JobStatusSync, JobVoteReminder}

var (
	jobLastSuccess = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scheduler_job_last_success_timestamp_seconds",
		Help: "Unix timestamp do último sucesso de cada job do scheduler.",
	}, []string{"job_name"})

	// Exposta como scheduler_job_failures_total (o client adiciona _total no
	// client_python da v1; aqui o sufixo é explícito para o nome final coincidir).
	jobFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "scheduler_job_failures_total",
		Help: "Total de execuções com falha de cada job do scheduler.",
	}, []string{"job_name"})
)

// RecordJobSuccess marca o heartbeat de liveness do job — deve ser chamada em
// todo término sem erro, mesmo quando o job não teve nada a fazer.
func RecordJobSuccess(jobName string) {
	jobLastSuccess.WithLabelValues(jobName).SetToCurrentTime()
}

// RecordJobFailure incrementa o contador de falhas do job.
func RecordJobFailure(jobName string) {
	jobFailures.WithLabelValues(jobName).Inc()
}

// InitJobMetrics inicializa as séries dos jobs no boot (gauge = agora,
// counter = 0), evitando staleness no `time() - max(...)` dos alertas e
// increase() sem série base logo após um restart.
func InitJobMetrics() {
	for _, job := range allJobs {
		jobLastSuccess.WithLabelValues(job).SetToCurrentTime()
		jobFailures.WithLabelValues(job).Add(0)
	}
}
