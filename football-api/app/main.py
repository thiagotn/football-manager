import time
from contextlib import asynccontextmanager

import structlog
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from apscheduler.triggers.cron import CronTrigger
from fastapi import FastAPI, Request, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, RedirectResponse

from app.api.v1.router import api_router
from app.core.config import get_settings
from app.services.recurrence import run_recurrence_job

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

    scheduler = AsyncIOScheduler(timezone="America/Sao_Paulo")
    scheduler.add_job(run_recurrence_job, CronTrigger(hour=7, minute=0))
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
    async def log_requests(request: Request, call_next):
        start = time.time()
        response = await call_next(request)
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
    return app


app = create_app()
