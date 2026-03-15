import re
import ssl
from pathlib import Path
from urllib.parse import urlparse, urlencode, parse_qs, urlunparse

import asyncpg
import structlog

logger = structlog.get_logger()

MIGRATIONS_DIR = Path(__file__).parent.parent.parent / "migrations"


async def run_migrations(database_url: str) -> None:
    """Apply any pending SQL migrations, tracking state in schema_migrations table."""
    dsn = re.sub(r"^postgresql\+asyncpg://", "postgresql://", database_url)

    # Extrai ?ssl=require da URL e passa como argumento SSL para o asyncpg
    connect_kwargs: dict = {}
    parsed = urlparse(dsn)
    qs = parse_qs(parsed.query)
    if "ssl" in qs:
        ssl_value = qs.pop("ssl")[0]
        if ssl_value in ("require", "true", "1"):
            ctx = ssl.create_default_context()
            ctx.check_hostname = False
            ctx.verify_mode = ssl.CERT_NONE
            connect_kwargs["ssl"] = ctx
        new_query = urlencode({k: v[0] for k, v in qs.items()})
        dsn = urlunparse(parsed._replace(query=new_query))

    conn = await asyncpg.connect(dsn, **connect_kwargs)
    try:
        await conn.execute("""
            CREATE TABLE IF NOT EXISTS schema_migrations (
                filename TEXT PRIMARY KEY,
                applied_at TIMESTAMPTZ DEFAULT NOW()
            )
        """)

        # Lock exclusivo para evitar race condition entre workers simultâneos
        await conn.execute("SELECT pg_advisory_lock(20260101)")

        try:
            applied = {
                row["filename"]
                for row in await conn.fetch("SELECT filename FROM schema_migrations")
            }

            pending = sorted(
                p for p in MIGRATIONS_DIR.glob("*.sql") if p.name not in applied
            )

            if not pending:
                logger.info("migrations_up_to_date")
                return

            for path in pending:
                logger.info("migration_applying", file=path.name)
                await conn.execute(path.read_text())
                await conn.execute(
                    "INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING",
                    path.name,
                )
                logger.info("migration_applied", file=path.name)
        finally:
            await conn.execute("SELECT pg_advisory_unlock(20260101)")

    finally:
        await conn.close()
