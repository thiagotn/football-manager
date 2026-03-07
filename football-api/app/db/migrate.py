import re
from pathlib import Path

import asyncpg
import structlog

logger = structlog.get_logger()

MIGRATIONS_DIR = Path(__file__).parent.parent.parent / "migrations"


async def run_migrations(database_url: str) -> None:
    """Apply any pending SQL migrations, tracking state in schema_migrations table."""
    dsn = re.sub(r"^postgresql\+asyncpg://", "postgresql://", database_url)

    conn = await asyncpg.connect(dsn)
    try:
        await conn.execute("""
            CREATE TABLE IF NOT EXISTS schema_migrations (
                filename TEXT PRIMARY KEY,
                applied_at TIMESTAMPTZ DEFAULT NOW()
            )
        """)

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
                "INSERT INTO schema_migrations (filename) VALUES ($1)", path.name
            )
            logger.info("migration_applied", file=path.name)

    finally:
        await conn.close()
