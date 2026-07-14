# PRD 049 — Scheduler resiliente para jobs agendados

**Status**: 📋 Proposto — implementar **após a migração do projeto para o homelab**

---

## Contexto — Incidente de 2026-07-04 a 2026-07-06

O rachão semanal do Futebol GQC (#17, previsto para 09/07) não foi criado pelo job de recorrência das 07h. Diagnóstico completo do incidente:

1. **04/07 ~19h–21h BRT** — o pooler do Supabase (Supavisor em *session mode*, limite de **15 clientes**) saturou. Todos os jobs do APScheduler passaram a falhar com `EMAXCONNSESSION: max clients reached in session mode - max clients are limited to pool_size: 15`.
2. As falhas ocorriam **em dobro a cada tick** — produção roda `uvicorn --workers 2` e o `lifespan` sobe **um APScheduler por worker**: todos os jobs (recorrência, status sync, lembrete de votação) e os pushes correspondentes executam duplicados o tempo todo.
3. **04/07 21h00 BRT (05/07 00:00 UTC)** — último evento de job nos logs. A partir daí, **silêncio total dos schedulers dos dois workers** por ~43h: nenhum `recurrence_job_done` (logado incondicionalmente), nenhum `vote_reminder_job_*`, nada. A API HTTP continuou servindo normalmente.
4. Causa provável do travamento permanente: conexões do pool mortas durante o incidente ficaram penduradas em `await` sem timeout (`pool_pre_ping`/queries não têm `command_timeout`); com `max_instances=1` (default do APScheduler), o job travado bloqueia silenciosamente todas as execuções seguintes — e os warnings de "skipped" do APScheduler não aparecem nos logs (logger sem handler configurado).
5. **Remediação aplicada em 06/07**: `docker restart football-api`. Schedulers voltaram; a partida atrasada é criada na próxima execução das 07h.
6. **Reincidência em 07/07** (noite seguinte ao restart) — mesmo padrão: saturação `EMAXCONNSESSION` no pico ~20h BRT, scheduler mudo a partir de 00:00 UTC, job das 07h não rodou. **Causa raiz confirmada por inspeção de rede** (`nsenter ... ss -tn`): o próprio `football-api` segurava **15/15 conexões do pooler às 07h, sem tráfego** — 2 workers × 2 engines (`session.py` + `core/database.py`) × `pool_size=5` = até 20 conexões idle mantidas indefinidamente (sem `pool_recycle`), acima do teto de 15 do Supavisor. Agravante: `run_recurrence_job`/`run_status_sync_job`/`run_vote_reminder_job` chamam `get_session_factory()` sem argumento, criando **um engine + pool novos a cada execução** (vote_reminder: a cada 5 min × 2 workers) em vez de reusar o singleton — engines async abandonados não fecham conexões asyncpg de forma confiável no GC. Remediação manual: restart + `docker exec football-api python3 -c "import asyncio; from app.services.recurrence import run_recurrence_job; asyncio.run(run_recurrence_job())"` (executar UMA vez, fora das 07h) → partida GQC #17 criada. **Enquanto este PRD não for implementado, a falha tende a reincidir toda noite.**

O sintoma para o usuário final: rachão não criado, lembretes de votação não enviados, partidas não fechadas/transicionadas automaticamente — tudo silenciosamente.

---

## Objetivo

Garantir que os jobs agendados (recorrência, status sync, lembrete de votação) sejam:
1. **Únicos** — uma execução por tick, independente do número de workers
2. **Resilientes** — falha ou travamento em um tick não impede os ticks seguintes
3. **Observáveis** — travamento/parada dos jobs gera alerta em minutos, não dias

---

## Escopo

### 1. Scheduler único (elimina duplicação)

Hoje: `Dockerfile` prod usa `--workers 2` e `app/main.py` sobe o scheduler no lifespan de cada worker.

Opções (decidir na implementação, considerando a infra do homelab):

- **A — Processo dedicado** (recomendada): container/serviço separado (`football-scheduler`) que roda apenas o APScheduler com os 3 jobs, sem servir HTTP. Workers da API deixam de subir scheduler. Isola falha dos jobs da API e vice-versa.
- **B — Lock de líder**: advisory lock do Postgres (`pg_try_advisory_lock`) no startup; só o worker que obtém o lock sobe o scheduler. Menos infra, porém mantém acoplamento com o ciclo de vida dos workers.

Efeito colateral corrigido: pushes duplicados para os jogadores (hoje cada notificação de job é enviada 2×).

### 2. Timeouts e higiene de conexões no engine

Em `app/db/session.py` (e no engine secundário de `app/core/database.py`):

- `connect_args={"command_timeout": <ex.: 30>}` no asyncpg — query travada falha em vez de pendurar o job para sempre
- `pool_recycle` (ex.: 300s) — recicla conexões antigas antes que o Supavisor as descarte silenciosamente
- Revisar `pool_size`/`max_overflow` considerando o orçamento total de conexões do banco: hoje `2 workers × (5+5)` + engine secundário + `football-api-go` + `football-mcp` disputam 15 conexões do pooler. No homelab (Postgres próprio, sem Supavisor) o teto muda — recalcular.
- Envolver o corpo de cada job em `asyncio.wait_for(..., timeout=<ex.: 120s>)` como cinto de segurança adicional.

### 3. Configuração do APScheduler

- `misfire_grace_time` explícito (ex.: 3600s para o job das 07h) — se o processo estiver ocupado/reiniciando no horário, o job ainda executa ao voltar
- `coalesce=True` — execuções perdidas acumuladas viram uma só
- Configurar logging do APScheduler para stdout (structlog/logging bridge) — hoje warnings como "maximum number of running instances reached" são perdidos

### 4. Observabilidade / heartbeat

> **✅ Parcialmente implementado em 2026-07-13** (antecipado após a reincidência de 12-13/07):
> - Métricas Prometheus `scheduler_job_last_success_timestamp_seconds{job_name,pid}` e `scheduler_job_failures_total{job_name,pid}` em `football-api/app/core/job_metrics.py`, registradas inline nos 3 jobs e inicializadas no lifespan
> - 3 regras de alerta Grafana (grupo "Scheduler" em `football-api/monitoring/grafana/provisioning/alerting/rules.yml`): scheduler mudo (> 20min sem vote_reminder, critical), recorrência > 26h (critical), jobs falhando (warning) — notificação via contact point Telegram existente
> - Painéis no dashboard `apis.json` (row "Scheduler v1"): idade do último sucesso por job + falhas
>
> Pendentes deste item: log `*_job_done` incondicional e retenção de logs.

- ~~Cada job grava timestamp da última execução bem-sucedida (tabela `job_heartbeats` ou métrica Prometheus `job_last_success_timestamp`)~~ ✅
- ~~Alerta (Uptime Kuma ou alertmanager/Grafana já existentes na VPS) se `recurrence` não roda há > 25h ou `status_sync` há > 2h~~ ✅ (via Grafana unified alerting)
- Logar `*_job_done` incondicionalmente em **todos** os jobs (hoje `vote_reminder` e `status_sync` são silenciosos em sucesso, o que dificultou o diagnóstico)
- Aumentar retenção de logs do container (`logging.options.max-size/max-file` no compose prod) — no incidente, a janela de ~2 dias quase apagou as evidências

### 5. Paridade v1/v2

Se a API Go (v2) assumir jobs agendados no futuro, aplicar as mesmas garantias (job único, timeouts, heartbeat). Atualizar PRD 044 §17 se aplicável.

---

## Fora de escopo

- Migração para fila/worker externo (Celery, Temporal, river etc.) — overkill para 3 cron jobs
- Retry automático de pushes falhados

---

## Critérios de aceite

1. Com N workers, cada job executa exatamente 1× por tick (verificável nos logs: 1 evento por tick, não 2)
2. Matar a conexão do banco durante um job não impede a execução do tick seguinte (falha logada + recuperação automática)
3. Parada do scheduler gera alerta em ≤ 30 min
4. `docker logs` de produção cobre ≥ 7 dias de eventos de jobs
