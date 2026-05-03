import time
from contextlib import asynccontextmanager

import structlog
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from apscheduler.triggers.cron import CronTrigger
from fastapi import FastAPI, Request, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, RedirectResponse
from prometheus_fastapi_instrumentator import Instrumentator

from app.api.v1.router import api_router
from app.core.config import get_settings
from app.db.migrate import run_migrations
from app.services.recurrence import run_recurrence_job, run_status_sync_job

logger = structlog.get_logger()


def setup_logging():
    settings = get_settings()
    import logging
    processors = [
        structlog.contextvars.merge_contextvars,
        structlog.processors.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
    ]
    if settings.is_prod:
        processors.append(structlog.processors.JSONRenderer())
    else:
        processors.append(structlog.dev.ConsoleRenderer(colors=True))
    structlog.configure(
        processors=processors,
        wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
    )


@asynccontextmanager
async def lifespan(app: FastAPI):
    app.state.start_time = time.time()

    await run_migrations(get_settings().database_url)

    scheduler = AsyncIOScheduler(timezone="America/Sao_Paulo")
    # 07h: fecha partidas passadas + cria próximos rachões por recorrência
    scheduler.add_job(run_recurrence_job, CronTrigger(hour=7, minute=0))
    # A cada hora (:30): fecha partidas passadas e transiciona para IN_PROGRESS
    # Roda em :30 para não coincidir com o job das 07:00
    scheduler.add_job(run_status_sync_job, CronTrigger(minute=30))
    scheduler.start()
    logger.info("api_started", version=get_settings().app_version)

    yield

    scheduler.shutdown()
    logger.info("api_stopped")


def create_app() -> FastAPI:
    setup_logging()
    settings = get_settings()

    app = FastAPI(
        title=settings.app_name,
        version=settings.app_version,
        description="""
## API de Gestão de Grupos de Futebol

### Funcionalidades
- 👥 **Grupos** — Crie e gerencie grupos de futebol
- ⚽ **Jogadores** — Cadastro completo com perfis (admin/jogador)
- 📅 **Partidas** — Agendamento com link único para compartilhamento
- ✅ **Presenças** — Confirmação/recusa de participação
- 📨 **Convites** — Links de convite com expiração em 30 minutos

### Autenticação
Use `POST /api/v1/auth/login` com seu WhatsApp e senha para obter o token Bearer.
        """,
        lifespan=lifespan,
        docs_url="/docs",
        redoc_url="/redoc",
    )

    app.add_middleware(
        CORSMiddleware,
        allow_origins=settings.cors_origins_list,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    @app.middleware("http")
    async def set_secure_headers(request: Request, call_next):
        response = await call_next(request)
        response.headers["X-Frame-Options"] = "DENY"
        response.headers["X-Content-Type-Options"] = "nosniff"
        response.headers["X-XSS-Protection"] = "1; mode=block"
        response.headers["Referrer-Policy"] = "strict-origin-when-cross-origin"
        if settings.is_prod:
            response.headers["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains"
        return response

    @app.middleware("http")
    async def log_requests(request: Request, call_next):
        start = time.time()
        response = await call_next(request)
        # Não loga /metrics e /health para não poluir os logs com scrapes do Prometheus
        if request.url.path not in ("/metrics", "/health"):
            duration = round((time.time() - start) * 1000, 1)
            logger.info(
                "request",
                method=request.method,
                path=request.url.path,
                status=response.status_code,
                ms=duration,
            )
        return response

    @app.get("/", include_in_schema=False)
    async def root():
        return RedirectResponse(url="/docs")

    @app.get("/health", tags=["infra"])
    async def health():
        uptime = round(time.time() - app.state.start_time, 1)
        return {"status": "ok", "uptime_seconds": uptime, "version": settings.app_version}

    app.include_router(api_router)

    # Expõe /metrics para o Prometheus (acessível apenas internamente via Docker network)
    Instrumentator(
        should_group_status_codes=False,
        excluded_handlers=["/metrics", "/health"],
    ).instrument(app).expose(app, endpoint="/metrics", include_in_schema=False)

    return app


app = create_app()
