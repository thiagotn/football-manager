from datetime import datetime, timedelta, time as dt_time
from zoneinfo import ZoneInfo

BRT = ZoneInfo("America/Sao_Paulo")

# Pontos por posição
POINTS: dict[int, int] = {1: 10, 2: 8, 3: 6, 4: 4, 5: 2}

VOTING_OPEN_DELAY = timedelta(minutes=20)
VOTING_DURATION   = timedelta(hours=24)


def voting_window(match) -> tuple[datetime, datetime]:
    """Retorna (opens_at, closes_at) em BRT."""
    end_t = match.end_time if match.end_time else dt_time(23, 59)
    end_dt = datetime.combine(match.match_date, end_t).replace(tzinfo=BRT)
    opens_at  = end_dt + VOTING_OPEN_DELAY
    closes_at = opens_at + VOTING_DURATION
    return opens_at, closes_at


def voting_status(match) -> str:
    """'not_open' | 'open' | 'closed'"""
    now = datetime.now(BRT)
    opens_at, closes_at = voting_window(match)
    if now < opens_at:
        return "not_open"
    if now <= closes_at:
        return "open"
    return "closed"


def time_until(target: datetime) -> str:
    """Retorna string legível 'Xh Ymin' até o target."""
    diff = target - datetime.now(BRT)
    total = max(0, int(diff.total_seconds()))
    h, rem = divmod(total, 3600)
    m = rem // 60
    if h > 0:
        return f"{h}h {m}min" if m else f"{h}h"
    return f"{m}min"
