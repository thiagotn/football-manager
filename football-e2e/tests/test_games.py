"""
E2E — jogos do rachao.app depois da mudança de casa (PRD 054).

Paciência, Tetris 3D e Enche Aí! vivem em games.thiagotn.com. No rachao.app
sobraram três redirects permanentes, para não quebrar link salvo nem PWA
instalado; o item Conta › Passatempo passou a apontar direto para o hub.

Os testes do jogo em si estão em `test_paciencia.py`, `test_tetris.py` e
`test_enche_ai.py`, que rodam contra o app `games/`.
"""

import pytest
from playwright.sync_api import Page

GAMES_URL = "https://games.thiagotn.com"

REDIRECTS = [
    ("/games", GAMES_URL),
    ("/paciencia", f"{GAMES_URL}/paciencia/"),
    ("/tetris", f"{GAMES_URL}/tetris/"),
]


@pytest.mark.parametrize(("path", "destino"), REDIRECTS)
def test_rota_antiga_redireciona_para_o_novo_dominio(page: Page, path: str, destino: str):
    """301 permanente, sem depender de o games.thiagotn.com estar no ar."""
    response = page.request.get(path, max_redirects=0)
    assert response.status == 301
    assert response.headers["location"] == destino
