"""
Testes unitários — routers/ranking.py

Regras de negócio cobertas:
- GET /api/v1/ranking — defaults (sem params, type=top) → 200 todos os tempos
- GET /api/v1/ranking?year=2026&month=3 → 200 mês específico
- GET /api/v1/ranking?year=2026 → 200 ano completo
- GET /api/v1/ranking?month=3 (sem year) → 422
- year fora do range (< 2024) → 422
- month fora do range (0 ou 13) → 422
- type inválido → 422
- GET /api/v1/ranking?type=flop → 200
- itens top possuem os campos corretos
- itens flop possuem os campos corretos
"""
from unittest.mock import AsyncMock

import pytest


# ── GET /api/v1/ranking ───────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_ranking_no_params_returns_all_time(api_client, mocker):
    """Sem parâmetros retorna todos os tempos (year=None, month=None)."""
    mocker.patch(
        "app.api.v1.routers.ranking.RankingRepository.get_top",
        new=AsyncMock(return_value=[]),
    )

    response = await api_client.get("/api/v1/ranking")

    assert response.status_code == 200
    data = response.json()
    assert data["year"] is None
    assert data["month"] is None
    assert data["type"] == "top"
    assert data["items"] == []


@pytest.mark.asyncio
async def test_get_ranking_year_and_month_returns_200(api_client, mocker):
    """year + month retorna ranking do mês específico."""
    mocker.patch(
        "app.api.v1.routers.ranking.RankingRepository.get_top",
        new=AsyncMock(return_value=[]),
    )

    response = await api_client.get("/api/v1/ranking?year=2026&month=3")

    assert response.status_code == 200
    data = response.json()
    assert data["year"] == 2026
    assert data["month"] == 3


@pytest.mark.asyncio
async def test_get_ranking_year_only_returns_200(api_client, mocker):
    """year sem month retorna ranking do ano completo."""
    mocker.patch(
        "app.api.v1.routers.ranking.RankingRepository.get_top",
        new=AsyncMock(return_value=[]),
    )

    response = await api_client.get("/api/v1/ranking?year=2026")

    assert response.status_code == 200
    data = response.json()
    assert data["year"] == 2026
    assert data["month"] is None


@pytest.mark.asyncio
async def test_get_ranking_month_without_year_returns_422(api_client):
    """month sem year deve retornar 422."""
    response = await api_client.get("/api/v1/ranking?month=3")

    assert response.status_code == 422


@pytest.mark.asyncio
async def test_get_ranking_year_below_range_returns_422(api_client):
    """year < 2024 deve retornar 422 (validação FastAPI)."""
    response = await api_client.get("/api/v1/ranking?year=2023")

    assert response.status_code == 422


@pytest.mark.asyncio
async def test_get_ranking_month_below_range_returns_422(api_client):
    """month=0 deve retornar 422."""
    response = await api_client.get("/api/v1/ranking?year=2026&month=0")

    assert response.status_code == 422


@pytest.mark.asyncio
async def test_get_ranking_month_above_range_returns_422(api_client):
    """month=13 deve retornar 422."""
    response = await api_client.get("/api/v1/ranking?year=2026&month=13")

    assert response.status_code == 422


@pytest.mark.asyncio
async def test_get_ranking_invalid_type_returns_422(api_client):
    """type inválido deve retornar 422."""
    response = await api_client.get("/api/v1/ranking?type=best")

    assert response.status_code == 422


@pytest.mark.asyncio
async def test_get_ranking_flop_returns_200(api_client, mocker):
    """type=flop retorna 200."""
    mocker.patch(
        "app.api.v1.routers.ranking.RankingRepository.get_flop",
        new=AsyncMock(return_value=[]),
    )

    response = await api_client.get("/api/v1/ranking?type=flop")

    assert response.status_code == 200
    data = response.json()
    assert data["type"] == "flop"
    assert data["items"] == []


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

    response = await api_client.get("/api/v1/ranking?year=2026&month=3&type=top")

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

    response = await api_client.get("/api/v1/ranking?type=flop")

    assert response.status_code == 200
    data = response.json()
    assert len(data["items"]) == 1
    item = data["items"][0]
    assert item["position"] == 1
    assert item["name"] == "Roberto Carlos"
    assert item["nickname"] is None
    assert item["total_flop_votes"] == 14
