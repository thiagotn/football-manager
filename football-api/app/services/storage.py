"""Supabase Storage — wrapper para upload/remoção de avatares."""
import json

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


def extract_storage_path(avatar_url: str) -> str | None:
    """Extrai o path relativo do arquivo a partir da URL pública do Supabase Storage.

    Ex: ".../object/public/avatars/uuid-token.webp" → "uuid-token.webp"
    Retorna None se a URL não corresponder ao padrão esperado.
    """
    marker = f"/public/{BUCKET}/"
    idx = avatar_url.find(marker)
    if idx == -1:
        return None
    return avatar_url[idx + len(marker):]


async def upload_avatar(player_id: str, webp_data: bytes, token: str) -> str:
    """Faz upload do avatar (WebP) para o Supabase Storage.

    O nome do arquivo inclui um token aleatório para evitar enumeração por player_id.
    Retorna a URL pública do arquivo.
    Lança RuntimeError se o Storage não estiver configurado.
    """
    if not _is_configured():
        raise RuntimeError("Supabase Storage não configurado (SUPABASE_URL / SUPABASE_SERVICE_ROLE_KEY).")

    path = f"{player_id}-{token}.webp"
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


async def delete_avatar_by_url(avatar_url: str) -> None:
    """Remove o avatar do Supabase Storage a partir da URL pública armazenada.

    Silencioso se o Storage não estiver configurado ou o arquivo não existir.
    """
    if not _is_configured():
        return

    path = extract_storage_path(avatar_url)
    if not path:
        logger.warning("avatar_delete_invalid_url", url=avatar_url)
        return

    delete_url = f"{_base_url()}/storage/v1/object/{BUCKET}"

    async with httpx.AsyncClient(timeout=10) as client:
        resp = await client.request(
            "DELETE",
            delete_url,
            content=json.dumps({"prefixes": [path]}),
            headers={**_headers(), "Content-Type": "application/json"},
        )
        if resp.status_code not in (200, 204, 400):
            logger.warning("storage_delete_warning", status=resp.status_code, body=resp.text)

    logger.info("avatar_deleted", path=path)
