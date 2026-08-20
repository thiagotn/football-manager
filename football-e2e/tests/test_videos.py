"""
E2E — vídeos curtos verticais por rachão (PRD 052, feature flag experimental).

Upload + transcodificação reais são inviáveis no CI (worker/ffmpeg fora do
stack de teste), então estes testes mockam os endpoints de vídeos via
`page.route` (mesmo padrão do mock version-agnostic do teste de sessão
expirada) e validam o comportamento do frontend:
  - card "Vídeos" na página da partida gateado por `group_videos_enabled`;
  - feed em /match/[hash]/videos (vídeo ready, estado vazio, processando);
  - página /admin/videos (real, sem mock).
"""

import json

import pytest
from playwright.sync_api import Page, expect

FAKE_HASH = "e2evideohash"

FAKE_MATCH = {
    "id": "11111111-1111-1111-1111-111111111111",
    "group_id": "22222222-2222-2222-2222-222222222222",
    "number": 1,
    "hash": FAKE_HASH,
    "match_date": "2026-08-20",
    "start_time": "19:00:00",
    "end_time": "20:00:00",
    "location": "Quadra E2E",
    "address": None,
    "court_type": None,
    "players_per_team": 5,
    "max_players": None,
    "notes": None,
    "status": "closed",
    "vote_open_delay_minutes": 0,
    "vote_duration_hours": 24,
    "created_at": "2026-08-20T10:00:00Z",
    "updated_at": "2026-08-20T10:00:00Z",
    "attendances": [],
    "confirmed_count": 0,
    "declined_count": 0,
    "pending_count": 0,
    "group_name": "Grupo E2E Vídeos",
    "group_timezone": "America/Sao_Paulo",
    "group_per_match_amount": None,
    "group_monthly_amount": None,
    "group_is_public": True,
    "group_voting_enabled": False,
    "group_videos_enabled": True,
}

READY_VIDEO = {
    "id": "33333333-3333-3333-3333-333333333333",
    "match_id": FAKE_MATCH["id"],
    "status": "ready",
    "video_url": "https://cdn.rachao.app/videos/fake/ready.mp4",
    "poster_url": "https://cdn.rachao.app/videos/fake/ready.webp",
    "duration_seconds": 42.5,
    "created_at": "2026-08-20T12:00:00Z",
    "uploader": {
        "id": "44444444-4444-4444-4444-444444444444",
        "name": "Jogador E2E",
        "nickname": None,
        "avatar_url": None,
    },
}


def _fulfill_json(route, payload):
    route.fulfill(
        status=200,
        content_type="application/json",
        body=json.dumps(payload),
    )


def _mock_match(page: Page, videos_enabled: bool):
    """Mocka o payload público da partida (version-agnostic: /api/v1 ou /api/v2)."""
    match = {**FAKE_MATCH, "group_videos_enabled": videos_enabled}
    page.route(
        f"**/matches/public/{FAKE_HASH}",
        lambda route: _fulfill_json(route, match),
    )
    # Chamadas secundárias da página da partida — respostas vazias inofensivas
    page.route(
        f"**/matches/public/{FAKE_HASH}/player-stats",
        lambda route: _fulfill_json(route, {"stats": []}),
    )
    page.route(
        "**/matches/*/teams",
        lambda route: _fulfill_json(route, {"teams": [], "reserves": []}),
    )


def _mock_videos(page: Page, videos, videos_enabled=True, can_upload=False):
    payload = {
        "videos": videos,
        "count": len([v for v in videos if v["status"] != "failed"]),
        "max_videos": 10,
        "can_upload": can_upload,
        "videos_enabled": videos_enabled,
    }
    page.route(
        f"**/matches/public/{FAKE_HASH}/videos",
        lambda route: _fulfill_json(route, payload),
    )


@pytest.fixture
def anon_page(browser):
    ctx = browser.new_context()
    page = ctx.new_page()
    yield page
    ctx.close()


# ── Card na página da partida ────────────────────────────────────────────────


def test_card_videos_oculto_sem_flag(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=False)
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}")
    anon_page.wait_for_load_state("networkidle")
    expect(anon_page.locator(f"a[href='/match/{FAKE_HASH}/videos']")).to_have_count(0)


def test_card_videos_visivel_com_flag(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}")
    anon_page.wait_for_load_state("networkidle")
    card = anon_page.locator(f"a[href='/match/{FAKE_HASH}/videos']")
    expect(card).to_be_visible()
    expect(card).to_contain_text("Vídeos")


# ── Feed /match/[hash]/videos ────────────────────────────────────────────────


def test_feed_renderiza_video_ready(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    _mock_videos(anon_page, [READY_VIDEO])
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/videos")
    anon_page.wait_for_load_state("networkidle")

    video_el = anon_page.locator("video")
    expect(video_el).to_have_count(1)
    expect(video_el).to_have_attribute("poster", READY_VIDEO["poster_url"])
    expect(anon_page.get_by_text("Jogador E2E")).to_be_visible()
    expect(anon_page.get_by_text("1/10")).to_be_visible()


def test_feed_estado_vazio(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    _mock_videos(anon_page, [])
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/videos")
    anon_page.wait_for_load_state("networkidle")
    expect(anon_page.get_by_text("Nenhum vídeo ainda.")).to_be_visible()
    expect(anon_page.locator("video")).to_have_count(0)


def test_feed_sem_botao_upload_para_anonimo(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    _mock_videos(anon_page, [READY_VIDEO], can_upload=False)
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/videos")
    anon_page.wait_for_load_state("networkidle")
    expect(anon_page.get_by_role("button", name="Enviar vídeo")).to_have_count(0)


def test_feed_botao_upload_para_autorizado(admin_page: Page, base_url):
    _mock_match(admin_page, videos_enabled=True)
    _mock_videos(admin_page, [], can_upload=True)
    admin_page.goto(f"{base_url}/match/{FAKE_HASH}/videos")
    admin_page.wait_for_load_state("networkidle")
    expect(admin_page.get_by_role("button", name="Enviar vídeo")).to_be_visible()


# ── Admin /admin/videos (real, sem mock) ─────────────────────────────────────


def test_admin_videos_page_carrega(admin_page: Page):
    admin_page.goto("/admin/videos")
    admin_page.wait_for_load_state("networkidle")
    expect(admin_page.get_by_role("heading", name="Vídeos (experimental)")).to_be_visible()
    # Tabela renderizada (com usuários ou estado vazio) — sem erro de carga
    expect(admin_page.locator(".alert-error")).to_have_count(0)
