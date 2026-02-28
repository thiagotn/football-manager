"""
Cria o primeiro superadmin ao iniciar a aplicação, se não existir.
"""
import structlog
from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import get_settings
from app.core.security import hash_password
from app.models.player import Player
from app.models.user import User

logger = structlog.get_logger()


async def seed_first_admin(db: AsyncSession) -> None:
    settings = get_settings()
    count_res = await db.execute(select(func.count(User.id)))
    if count_res.scalar_one() > 0:
        return  # already has users

    logger.info("seeding_first_admin", email=settings.first_admin_email)

    user = User(
        email=settings.first_admin_email,
        password_hash=hash_password(settings.first_admin_password),
        name=settings.first_admin_name,
        is_superadmin=True,
    )
    db.add(user)
    await db.flush()

    player = Player(
        user_id=user.id,
        name=settings.first_admin_name,
        whatsapp="5500000000000",  # placeholder
    )
    db.add(player)
    await db.commit()
    logger.info("first_admin_created", email=settings.first_admin_email)
