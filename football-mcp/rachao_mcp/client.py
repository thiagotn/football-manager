import os

import httpx

from rachao_mcp.auth import get_api_url, get_token


class RachaoClient:
    async def _request(self, method: str, path: str, **kwargs) -> list | dict:
        group_allowlist_raw = os.getenv("RACHAO_MCP_GROUP_ALLOWLIST", "")
        group_allowlist: set[str] | None = (
            set(group_allowlist_raw.split(",")) if group_allowlist_raw else None
        )

        if group_allowlist and "/groups/" in path:
            gid = path.split("/groups/")[1].split("/")[0]
            if gid not in group_allowlist:
                raise PermissionError(
                    f"Grupo {gid} não está na allowlist do MCP. "
                    f"Grupos permitidos: {group_allowlist}"
                )

        url = get_api_url() + path
        headers = {"Authorization": f"Bearer {get_token()}"}

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.request(method, url, headers=headers, **kwargs)
        except (httpx.ConnectError, httpx.TimeoutException) as exc:
            raise RuntimeError("API indisponível — verifique sua conexão") from exc

        if response.status_code == 401:
            raise PermissionError("Autenticação inválida — verifique RACHAO_TOKEN")
        if response.status_code == 403:
            raise PermissionError(f"Sem permissão: {response.text}")
        if response.status_code == 404:
            raise LookupError("Recurso não encontrado")
        if response.status_code >= 500:
            raise RuntimeError(f"API indisponível (HTTP {response.status_code})")
        if response.status_code >= 400:
            raise ValueError(f"Erro na requisição ({response.status_code}): {response.text}")

        return response.json()

    async def get(self, path: str, **kwargs) -> list | dict:
        return await self._request("GET", path, **kwargs)

    async def post(self, path: str, **kwargs) -> list | dict:
        return await self._request("POST", path, **kwargs)

    async def patch(self, path: str, **kwargs) -> list | dict:
        return await self._request("PATCH", path, **kwargs)


api = RachaoClient()
