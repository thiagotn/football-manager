"""
Testes unitários — routers/ranking.py

Regras de negócio cobertas:
- GET /api/v1/ranking — defaults (period=month, type=top) → 200 com lista vazia
- GET /api/v1/ranking?period=year&type=top → 200
- GET /api/v1/ranking?period=all&type=flop → 200
- period inválido → 422
- type inválido → 422

O endpoint é público (sem autenticação). O banco é mockado para retornar
listas vazias, verificando apenas estrutura da resposta e validação de params.
"""
from unittest.mock import AsyncMock

import pytest


# ── GET /api/v1/ranking ───────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_ranking_default_params_returns_200(api_client, mocker):
    """Defaults period=month, type=top retorna 200 com itens vazios."""
    mocker.patch(
        "app.api.v1.routers.ranking.RankingRepository.get_top",
        new=AsyncMock(return_value=[]),
    )

    response = await api_client.get("/api/v1/ranking")

    assert response.status_code == 200
    data = response.json()
    assert data["period"] == "month"
    assert data["type"] == "top"
    assert data["items"] == []


@pytest.mark.asyncio
async def test_get_ranking_year_top_returns_200(api_client, mocker):
    """period=year&type=top retorna 200."""
    mocker.patch(
        "app.api.v1.routers.ranking.RankingRepository.get_top",
        new=AsyncMock(return_value=[]),
    )

    response = await api_client.get("/api/v1/ranking?period=year&type=top")

    assert response.status_code == 200
    data = response.json()
    assert data["period"] == "year"
    assert data["type"] == "top"


@pytest.mark.asyncio
async def test_get_ranking_all_flop_returns_200(api_client, mocker):
    """period=all&type=flop retorna 200."""
    mocker.patch(
        "app.api.v1.routers.ranking.RankingRepository.get_flop",
        new=AsyncMock(return_value=[]),
    )

    response = await api_client.get("/api/v1/ranking?period=all&type=flop")

    assert response.status_code == 200
    data = response.json()
    assert data["period"] == "all"
    assert data["type"] == "flop"
    assert data["items"] == []


@pytest.mark.asyncio
async def test_get_ranking_invalid_period_returns_422(api_client):
    """period inválido deve retornar 422 (validação FastAPI)."""
    response = await api_client.get("/api/v1/ranking?period=week")

    assert response.status_code == 422


@pytest.mark.asyncio
async def test_get_ranking_invalid_type_returns_422(api_client):
    """type inválido deve retornar 422 (validação FastAPI)."""
    response = await api_client.get("/api/v1/ranking?type=best")

    assert response.status_code == 422


@pytest.mark.asyncio
async def test_get_ranking_top_returns_items_with_correct_fields(api_client, mocker):
    """Verifica que itens top possuem os campos corretos."""
    from uuid import uuid4
    player_id = uuid4()
    mocker.patch(
        "app.api.v1.routers.ranking.RankingRepository.get_top",
        new=AsyncMock(return_value=[
            {
                "position": 1,
                "player_id": player_id,
                "name": "Thiago Nunes",
                "nickname": "Thiagol",
                "avatar_url": None,
                "total_points": 312,
            }
        ]),
    )

    response = await api_client.get("/api/v1/ranking?period=all&type=top")

    assert response.status_code == 200
    data = response.json()
    assert len(data["items"]) == 1
    item = data["items"][0]
    assert item["position"] == 1
    assert item["name"] == "Thiago Nunes"
    assert item["nickname"] == "Thiagol"
    assert item["total_points"] == 312
    assert item["avatar_url"] is None


@pytest.mark.asyncio
async def test_get_ranking_flop_returns_items_with_correct_fields(api_client, mocker):
    """Verifica que itens flop possuem os campos corretos."""
    from uuid import uuid4
    player_id = uuid4()
    mocker.patch(
        "app.api.v1.routers.ranking.RankingRepository.get_flop",
        new=AsyncMock(return_value=[
            {
                "position": 1,
                "player_id": player_id,
                "name": "Roberto Carlos",
                "nickname": None,
                "avatar_url": None,
                "total_flop_votes": 14,
            }
        ]),
    )

    response = await api_client.get("/api/v1/ranking?period=month&type=flop")

    assert response.status_code == 200
    data = response.json()
    assert len(data["items"]) == 1
    item = data["items"][0]
    assert item["position"] == 1
    assert item["name"] == "Roberto Carlos"
    assert item["nickname"] is None
    assert item["total_flop_votes"] == 14
