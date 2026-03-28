"""
Testes unitários — routers/reviews.py

Regras de negócio cobertas:
- GET /reviews/me super admin → 403
- PUT /reviews/me super admin → 403
- GET /reviews/me sem avaliação existente → 404
- GET /reviews/summary não-admin → 403
- GET /reviews não-admin → 403
"""
from unittest.mock import AsyncMock

import pytest


# ── GET /reviews/me ───────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_my_review_admin_returns_403(admin_client):
    """Super admins não podem avaliar o app."""
    response = await admin_client.get("/api/v1/reviews/me")

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_get_my_review_not_found_returns_404(api_client, mocker):
    """Jogador sem avaliação registrada recebe 404."""
    mocker.patch(
        "app.api.v1.routers.reviews.ReviewRepository.get_by_player",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get("/api/v1/reviews/me")

    assert response.status_code == 404


# ── PUT /reviews/me ───────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_upsert_review_admin_returns_403(admin_client):
    """Super admins não podem submeter avaliações."""
    response = await admin_client.put(
        "/api/v1/reviews/me",
        json={"rating": 5, "comment": "Ótimo app"},
    )

    assert response.status_code == 403


# ── GET /reviews/summary ──────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_review_summary_non_admin_returns_403(api_client):
    """Resumo de avaliações é restrito a admins globais."""
    response = await api_client.get("/api/v1/reviews/summary")

    assert response.status_code == 403


# ── GET /reviews ──────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_list_reviews_non_admin_returns_403(api_client):
    """Listagem completa de avaliações é restrita a admins globais."""
    response = await api_client.get("/api/v1/reviews")

    assert response.status_code == 403


# ── PUT /reviews/me — happy path ─────────────────────────────────────────────


@pytest.mark.asyncio
async def test_upsert_review_success_returns_200(api_client, player_user, mocker):
    """Jogador pode criar ou atualizar sua avaliação do app."""
    from datetime import datetime
    from uuid import uuid4

    review = AsyncMock()
    review.id = uuid4()
    review.rating = 5
    review.comment = "Excelente app!"
    review.created_at = datetime(2026, 3, 1)
    review.updated_at = datetime(2026, 3, 1)

    mocker.patch(
        "app.api.v1.routers.reviews.ReviewRepository.upsert",
        new=AsyncMock(return_value=review),
    )

    response = await api_client.put(
        "/api/v1/reviews/me",
        json={"rating": 5, "comment": "Excelente app!"},
    )

    assert response.status_code == 200
    assert response.json()["rating"] == 5


# ── GET /reviews/me — happy path ─────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_my_review_success_returns_200(api_client, player_user, mocker):
    """Jogador com avaliação existente recebe dados da avaliação."""
    from datetime import datetime
    from uuid import uuid4

    review = AsyncMock()
    review.id = uuid4()
    review.rating = 4
    review.comment = "Muito bom"
    review.created_at = datetime(2026, 2, 1)
    review.updated_at = datetime(2026, 2, 1)

    mocker.patch(
        "app.api.v1.routers.reviews.ReviewRepository.get_by_player",
        new=AsyncMock(return_value=review),
    )

    response = await api_client.get("/api/v1/reviews/me")

    assert response.status_code == 200
    assert response.json()["rating"] == 4


# ── GET /reviews/summary — admin only ────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_review_summary_admin_returns_200(admin_client, mocker):
    """Admin global pode ver resumo das avaliações."""
    from app.schemas.review import ReviewSummaryResponse, DistributionEntry

    summary = ReviewSummaryResponse(
        average=4.2,
        total=50,
        distribution={
            "5": DistributionEntry(count=25, percent=50.0),
            "4": DistributionEntry(count=15, percent=30.0),
            "3": DistributionEntry(count=5, percent=10.0),
            "2": DistributionEntry(count=3, percent=6.0),
            "1": DistributionEntry(count=2, percent=4.0),
        },
    )
    mocker.patch(
        "app.api.v1.routers.reviews.ReviewRepository.get_summary",
        new=AsyncMock(return_value=summary),
    )

    response = await admin_client.get("/api/v1/reviews/summary")

    assert response.status_code == 200
    data = response.json()
    assert data["average"] == 4.2
    assert data["total"] == 50


# ── GET /reviews — admin only ────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_list_reviews_admin_returns_200(admin_client, mocker):
    """Admin global pode listar todas as avaliações."""
    mocker.patch(
        "app.api.v1.routers.reviews.ReviewRepository.list_all",
        new=AsyncMock(return_value=([], 0)),
    )

    response = await admin_client.get("/api/v1/reviews")

    assert response.status_code == 200
    data = response.json()
    assert data["items"] == []
    assert data["total"] == 0
