import re

from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine
from sqlalchemy.orm import DeclarativeBase

from app.core.config import get_settings


def _normalize_db_url(url: str) -> str:
    """Ensure the URL uses postgresql+asyncpg:// scheme."""
    return re.sub(r"^postgres(ql)?://", "postgresql+asyncpg://", url)


class Base(DeclarativeBase):
    pass


def get_engine():
    settings = get_settings()
    return create_async_engine(
        _normalize_db_url(settings.database_url),
        pool_size=10,
        max_overflow=20,
        echo=settings.debug,
    )


def get_session_factory(engine=None):
    if engine is None:
        engine = get_engine()
    return async_sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)


# Singleton instances
_engine = None
_session_factory = None


def _init():
    global _engine, _session_factory
    if _engine is None:
        _engine = get_engine()
        _session_factory = get_session_factory(_engine)


async def get_db() -> AsyncSession:  # type: ignore[misc]
    _init()
    async with _session_factory() as session:  # type: ignore[misc]
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise
