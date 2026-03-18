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
