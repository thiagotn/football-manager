"""Supabase Storage — wrapper para upload/remoção de avatares."""
import httpx
import structlog

from app.core.config import get_settings

logger = structlog.get_logger()

BUCKET = "avatars"


def _base_url() -> str:
    return get_settings().supabase_url.rstrip("/")


def _headers() -> dict:
    return {
        "Authorization": f"Bearer {get_settings().supabase_service_role_key}",
    }


def _is_configured() -> bool:
    s = get_settings()
    return bool(s.supabase_url and s.supabase_service_role_key)


async def upload_avatar(player_id: str, webp_data: bytes) -> str:
    """Faz upload do avatar (WebP) para o Supabase Storage.

    Retorna a URL pública do arquivo.
    Lança RuntimeError se o Storage não estiver configurado.
    """
    if not _is_configured():
        raise RuntimeError("Supabase Storage não configurado (SUPABASE_URL / SUPABASE_SERVICE_ROLE_KEY).")

    path = f"{player_id}.webp"
    upload_url = f"{_base_url()}/storage/v1/object/{BUCKET}/{path}"

    async with httpx.AsyncClient(timeout=20) as client:
        resp = await client.post(
            upload_url,
            content=webp_data,
            headers={
                **_headers(),
                "Content-Type": "image/webp",
                "x-upsert": "true",
            },
        )
        if resp.status_code not in (200, 201):
            logger.error("storage_upload_failed", status=resp.status_code, body=resp.text)
            raise RuntimeError(f"Erro ao salvar imagem no Storage (HTTP {resp.status_code}).")

    public_url = f"{_base_url()}/storage/v1/object/public/{BUCKET}/{path}"
    logger.info("avatar_uploaded", player_id=player_id, url=public_url)
    return public_url


async def delete_avatar(player_id: str) -> None:
    """Remove o avatar do Supabase Storage. Silencioso se não configurado ou não encontrado."""
    if not _is_configured():
        return

    path = f"{player_id}.webp"
    delete_url = f"{_base_url()}/storage/v1/object/{BUCKET}"

    async with httpx.AsyncClient(timeout=10) as client:
        resp = await client.delete(
            delete_url,
            json={"prefixes": [path]},
            headers={**_headers(), "Content-Type": "application/json"},
        )
        if resp.status_code not in (200, 204, 400):
            # 400 pode ocorrer se o arquivo não existir — ignorar
            logger.warning("storage_delete_warning", status=resp.status_code, body=resp.text)

    logger.info("avatar_deleted", player_id=player_id)
