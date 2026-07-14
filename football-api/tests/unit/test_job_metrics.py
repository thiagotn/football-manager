"""Testes unitários — app/core/job_metrics.py (heartbeat dos jobs, PRD 049 item 4).

Cenários cobertos:
- record_job_success seta o gauge para "agora"
- record_job_failure incrementa o counter
- init_job_metrics cria as séries dos 3 jobs (gauge > 0, counter presente)
- /metrics expõe ambas as métricas
"""
import os
import time

import pytest
from httpx import ASGITransport, AsyncClient
from prometheus_client import REGISTRY

from app.core.job_metrics import (
    ALL_JOBS,
    JOB_VOTE_REMINDER,
    init_job_metrics,
    record_job_failure,
    record_job_success,
)
from app.main import app

_PID = str(os.getpid())


def gauge_value(job_name: str):
    return REGISTRY.get_sample_value(
        "scheduler_job_last_success_timestamp_seconds",
        {"job_name": job_name, "pid": _PID},
    )


def counter_value(job_name: str):
    return REGISTRY.get_sample_value(
        "scheduler_job_failures_total",
        {"job_name": job_name, "pid": _PID},
    )


def test_record_job_success_sets_gauge_to_now():
    before = time.time()
    record_job_success(JOB_VOTE_REMINDER)
    value = gauge_value(JOB_VOTE_REMINDER)
    assert value is not None
    assert before <= value <= time.time()


def test_record_job_failure_increments_counter():
    base = counter_value(JOB_VOTE_REMINDER) or 0.0
    record_job_failure(JOB_VOTE_REMINDER)
    assert counter_value(JOB_VOTE_REMINDER) == base + 1


def test_init_job_metrics_creates_series_for_all_jobs():
    init_job_metrics()
    for name in ALL_JOBS:
        assert gauge_value(name) is not None
        assert gauge_value(name) > 0
        # Counter apenas "tocado" — precisa existir (>= 0), sem incrementar
        assert counter_value(name) is not None


@pytest.mark.asyncio
async def test_metrics_endpoint_exposes_series():
    # O lifespan não roda sob ASGITransport — inicializa as séries manualmente
    init_job_metrics()
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.get("/metrics")
    assert resp.status_code == 200
    assert "scheduler_job_last_success_timestamp_seconds" in resp.text
    assert "scheduler_job_failures_total" in resp.text
