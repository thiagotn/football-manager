"""Heartbeat Prometheus dos jobs do APScheduler (PRD 049, item 4 — observabilidade).

Métricas de *liveness*, não de resultado de negócio: `record_job_success` marca que
a execução completou, mesmo que nenhuma partida tenha sido criada/notificada.
Alimentam os alertas Grafana "Scheduler" (monitoring/grafana/provisioning/alerting/rules.yml),
que detectam o scheduler mudo após saturação do pooler Supabase (EMAXCONNSESSION).

O label `pid` distingue os 2 workers uvicorn de produção: cada scrape do Prometheus
cai num worker aleatório e, sem o label, o counter alternaria entre valores absolutos
de processos diferentes (resets perpétuos aos olhos do `increase()`).
"""
import os

from prometheus_client import Counter, Gauge

JOB_RECURRENCE = "recurrence"
JOB_STATUS_SYNC = "status_sync"
JOB_VOTE_REMINDER = "vote_reminder"
ALL_JOBS = (JOB_RECURRENCE, JOB_STATUS_SYNC, JOB_VOTE_REMINDER)

JOB_LAST_SUCCESS = Gauge(
    "scheduler_job_last_success_timestamp_seconds",
    "Unix timestamp do último sucesso de cada job do APScheduler",
    ["job_name", "pid"],
)

# Exposto como scheduler_job_failures_total (o client acrescenta o sufixo _total)
JOB_FAILURES = Counter(
    "scheduler_job_failures",
    "Total de falhas por job do APScheduler",
    ["job_name", "pid"],
)


def record_job_success(job_name: str) -> None:
    JOB_LAST_SUCCESS.labels(job_name=job_name, pid=str(os.getpid())).set_to_current_time()


def record_job_failure(job_name: str) -> None:
    JOB_FAILURES.labels(job_name=job_name, pid=str(os.getpid())).inc()


def init_job_metrics() -> None:
    """Cria as séries no startup (gauge = agora, counter = 0).

    Sem isto, o alerta de staleness operaria sobre métrica ausente até a primeira
    execução de cada job, e um counter nascido em N esconderia as N primeiras
    falhas do `increase()`.
    """
    for name in ALL_JOBS:
        record_job_success(name)
        JOB_FAILURES.labels(job_name=name, pid=str(os.getpid()))
