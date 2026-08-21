"""
E2E — vídeos curtos verticais por rachão (PRD 052, feature flag experimental).

Upload + transcodificação reais são inviáveis no CI (worker/ffmpeg fora do
stack de teste), então estes testes mockam os endpoints de vídeos via
`page.route` (mesmo padrão do mock version-agnostic do teste de sessão
expirada) e validam o comportamento do frontend:
  - card "Vídeos" na página da partida gateado por `group_videos_enabled`;
  - feed em /match/[hash]/feed (vídeo/foto ready, estado vazio, redirect de /videos);
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
    "media_type": "video",
    "video_url": "https://cdn.rachao.app/videos/fake/ready.mp4",
    "poster_url": "https://cdn.rachao.app/videos/fake/ready.webp",
    "duration_seconds": 42.5,
    "created_at": "2026-08-20T12:00:00Z",
    "like_count": 2,
    "liked_by_me": False,
    "uploader": {
        "id": "44444444-4444-4444-4444-444444444444",
        # Nome de uma palavra: o feed renderiza via playerDisplayName, que
        # mostra apenas o primeiro nome.
        "name": "JogadorE2E",
        "nickname": None,
        "avatar_url": None,
    },
}

READY_IMAGE = {
    "id": "66666666-6666-6666-6666-666666666666",
    "match_id": FAKE_MATCH["id"],
    "status": "ready",
    "media_type": "image",
    "video_url": None,
    "poster_url": "https://cdn.rachao.app/videos/fake/photo.jpg",
    "duration_seconds": None,
    "created_at": "2026-08-20T12:10:00Z",
    "like_count": 0,
    "liked_by_me": False,
    "uploader": {
        "id": "44444444-4444-4444-4444-444444444444",
        "name": "JogadorE2E",
        "nickname": None,
        "avatar_url": None,
    },
}

LIKERS = {
    "likers": [
        {
            "id": "55555555-5555-5555-5555-555555555555",
            "name": "CurtidorE2E",
            "nickname": None,
            "avatar_url": None,
            "created_at": "2026-08-20T12:30:00Z",
        }
    ],
    "count": 1,
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
    expect(anon_page.locator(f"a[href='/match/{FAKE_HASH}/feed']")).to_have_count(0)


def test_card_videos_visivel_com_flag(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}")
    anon_page.wait_for_load_state("networkidle")
    card = anon_page.locator(f"a[href='/match/{FAKE_HASH}/feed']")
    expect(card).to_be_visible()
    expect(card).to_contain_text("Feed")


# ── Feed /match/[hash]/videos ────────────────────────────────────────────────


def test_feed_renderiza_video_ready(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    _mock_videos(anon_page, [READY_VIDEO])
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/feed")
    anon_page.wait_for_load_state("networkidle")

    video_el = anon_page.locator("video")
    expect(video_el).to_have_count(1)
    expect(video_el).to_have_attribute("poster", READY_VIDEO["poster_url"])
    expect(anon_page.get_by_text("JogadorE2E")).to_be_visible()
    expect(anon_page.get_by_text("1/10")).to_be_visible()


def test_feed_estado_vazio(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    _mock_videos(anon_page, [])
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/feed")
    anon_page.wait_for_load_state("networkidle")
    expect(anon_page.get_by_text("Nenhum vídeo ou foto ainda.")).to_be_visible()
    expect(anon_page.locator("video")).to_have_count(0)


def test_feed_sem_botao_upload_para_anonimo(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    _mock_videos(anon_page, [READY_VIDEO], can_upload=False)
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/feed")
    anon_page.wait_for_load_state("networkidle")
    expect(anon_page.get_by_role("button", name="Enviar mídia")).to_have_count(0)


def test_feed_botao_upload_para_autorizado(admin_page: Page, base_url):
    _mock_match(admin_page, videos_enabled=True)
    _mock_videos(admin_page, [], can_upload=True)
    admin_page.goto(f"{base_url}/match/{FAKE_HASH}/feed")
    admin_page.wait_for_load_state("networkidle")
    # Estado vazio + can_upload: botão na barra superior E no CTA do slide vazio
    expect(admin_page.get_by_role("button", name="Enviar mídia").first).to_be_visible()


def test_feed_renderiza_foto(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    _mock_videos(anon_page, [READY_IMAGE, READY_VIDEO])
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/feed")
    anon_page.wait_for_load_state("networkidle")

    img = anon_page.locator(f"img[src='{READY_IMAGE['poster_url']}']")
    expect(img).to_be_visible()
    expect(anon_page.locator("video")).to_have_count(1)  # vídeo no slide seguinte
    expect(anon_page.get_by_text("2/10")).to_be_visible()


def test_redirect_videos_para_feed(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    _mock_videos(anon_page, [])
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/videos")
    anon_page.wait_for_load_state("networkidle")
    expect(anon_page).to_have_url(f"{base_url}/match/{FAKE_HASH}/feed")


def test_feed_botao_compartilhar_visivel(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    _mock_videos(anon_page, [READY_VIDEO])
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/feed")
    anon_page.wait_for_load_state("networkidle")
    expect(anon_page.get_by_role("button", name="Compartilhar")).to_be_visible()


# ── Curtidas ─────────────────────────────────────────────────────────────────


def test_like_incrementa_contador(admin_page: Page, base_url):
    _mock_match(admin_page, videos_enabled=True)
    _mock_videos(admin_page, [READY_VIDEO])
    admin_page.route(
        f"**/videos/{READY_VIDEO['id']}/like",
        lambda route: _fulfill_json(route, {"like_count": 3, "liked_by_me": True}),
    )
    admin_page.goto(f"{base_url}/match/{FAKE_HASH}/feed")
    admin_page.wait_for_load_state("networkidle")

    expect(admin_page.get_by_text("2", exact=True)).to_be_visible()  # contador inicial
    admin_page.get_by_role("button", name="Curtir").click()
    expect(admin_page.get_by_text("3", exact=True)).to_be_visible()  # otimista + resposta
    expect(admin_page.get_by_role("button", name="Remover curtida")).to_be_visible()


def test_curtidas_exibe_quem_curtiu(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    _mock_videos(anon_page, [READY_VIDEO])
    anon_page.route(
        f"**/videos/{READY_VIDEO['id']}/likes",
        lambda route: _fulfill_json(route, LIKERS),
    )
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/feed")
    anon_page.wait_for_load_state("networkidle")

    anon_page.get_by_role("button", name="Curtidas").click()
    expect(anon_page.get_by_text("CurtidorE2E")).to_be_visible()


def test_like_anonimo_pede_login(anon_page: Page, base_url):
    _mock_match(anon_page, videos_enabled=True)
    _mock_videos(anon_page, [READY_VIDEO])
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/feed")
    anon_page.wait_for_load_state("networkidle")

    anon_page.get_by_role("button", name="Curtir").click()
    expect(anon_page.get_by_text("Entre na sua conta para curtir.")).to_be_visible()


# ── Admin /admin/videos (real, sem mock) ─────────────────────────────────────


def test_admin_videos_page_carrega(admin_page: Page):
    admin_page.goto("/admin/videos")
    admin_page.wait_for_load_state("networkidle")
    expect(admin_page.get_by_role("heading", name="Vídeos (experimental)")).to_be_visible()
    # Tabela renderizada (com usuários ou estado vazio) — sem erro de carga
    expect(admin_page.locator(".alert-error")).to_have_count(0)
