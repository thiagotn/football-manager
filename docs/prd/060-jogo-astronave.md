# PRD 060 — Astronave

- **Status:** ✅ No ar em `games.thiagotn.com` desde 2026-11-27
- **Código:** [`games/src/lib/astronave/`](../../games/src/lib/astronave/) e [`games/src/routes/astronave/`](../../games/src/routes/astronave/) — estado corrente em [`games/CLAUDE.md`](../../games/CLAUDE.md)
- **Antecessores:** [PRD 055](055-jogo-cobrinha.md) (molde das abas, do diário e dos visuais) e [PRD 057](057-jogo-comilao.md) (física em passo fixo)

## Problema / Objetivo

O hub tem arcade de grade (Cobrinha), de labirinto (Comilão) e de encaixe (Tetris), mas
nenhum **jogo de nave**. O pedido: pilotar uma nave que desvia de meteoros e dos tiros dos
inimigos e atira para se defender, funcionando bem no celular, com a **escolha entre
botões na tela ou só o toque** — e, sem botões, a arena ocupando a tela.

É um jogo original do gênero "atire para cima": nave, meteoros e inimigos são desenhos
próprios, vetoriais. Os inimigos **não descem em bloco** até o jogador; entram, patrulham
e atiram, enquanto os meteoros caem o tempo todo.

## Decisões (2026-11-27)

| # | Decisão | Por quê |
|---|---|---|
| 1 | **A nave só anda na horizontal**, perto do pé da arena | É o que se controla bem com um dedo só; desviar vira leitura do que está caindo, não corrida pela tela |
| 2 | **Dois modos de controle, escolhidos nos Ajustes ou na pausa** — `touch` (padrão) e `pad` | Pedido explícito. O padrão é o toque porque é o que aproveita a tela |
| 3 | **No toque, o arraste é relativo e segurar atira** | A nave anda o tanto que o dedo anda, de onde ela estava: o dedo não precisa ficar em cima dela (e não a tapa). Um gesto só move e atira |
| 4 | **Campo lógico fixo de 100 × 170**, só escalado | O desafio do dia precisa ser o mesmo em qualquer aparelho. 100 × 170 enche um celular 20:9 no modo toque e ainda cabe com os botões |
| 5 | **`tick` com passo fixo de 1/60 s, mudando o estado no lugar** | A física não depende da taxa de quadros e o teste compara números exatos; são dezenas de objetos a 60 Hz, não vale recriar tudo a cada passo |
| 6 | **Dois sorteios no estado**: `rng` (o que nasce) e `rngDrop` (prêmios) | O que nasce no desafio do dia não pode depender de quem o jogador abateu |
| 7 | **Um em cada três meteoros mira a nave; o artilheiro mira ao atirar** | Ficar parado num canto não pode ser abrigo |
| 8 | **Nada se mexe antes do primeiro comando; sair do app pausa** | Mesmo motivo da Cobrinha: ninguém perde antes de pegar no controle |
| 9 | **Partida não é salva** | Voltar no meio de uma chuva de meteoros sem o dedo no lugar é perder vida de graça |

## Como ficou

| Arquivo | Conteúdo |
|---------|----------|
| `src/lib/astronave/engine.ts` | Regras puras: nave, tiros, meteoros (o grande racha em dois), batedor e artilheiro, prêmios (escudo, tiro duplo, vida), ondas, vidas, meta do dia |
| `src/lib/astronave/engine.test.ts` | 38 testes: movimento, cadência de tiro, colisões, dano e escudo, ondas, **determinismo do dia**, não usar `Math.random` e equilíbrio (parado: fim em < 1 min) |
| `src/lib/astronave/skins.ts` + teste | 6 naves (só cores sobre o mesmo contorno), liberadas por pontuação somada |
| `src/lib/astronave/render.ts` | Canvas 2D em unidades do campo: estrelas, meteoros com forma estável por id, inimigos, faíscas |
| `src/lib/astronave/storage.ts` | `games_astronave`: estatísticas, histórico (60), dias do desafio e ajustes (`controls`, `skin`) |
| `src/routes/astronave/+page.svelte` | Abas Início · Diário · Naves · Ajustes, a partida por cima, os dois modos de controle |

**Partida.** Três vidas. Meteoro pequeno (1 tiro, 10 pts) e grande (3 tiros, 25 pts, racha
em dois pequenos). Batedor (1 tiro, 30 pts, atira reto) e artilheiro (3 tiros, 60 pts, mira
na nave; aparece da onda 3 em diante). Limpar a onda vale `50 × onda` e dá 2 s de respiro.
Levar uma pancada custa uma vida e dá 2 s de invulnerabilidade (a nave pisca).

**Prêmios.** Caem de inimigos abatidos (14% batedor, 30% artilheiro): escudo (absorve uma
pancada), tiro duplo (10 s) e vida (só sai se faltar alguma).

**Controles.**
- *Só toque* (padrão): arrastar em qualquer ponto da arena move, segurar o dedo atira.
  Sem botões: num celular de 412 × 803 a arena fica com 400 × 680.
- *Botões*: ◀ ▶ à esquerda e **Atirar** à direita, embaixo da arena (372 × 634 no mesmo
  celular). Arrastar na arena continua valendo.
- Teclado nos dois: setas ou A/D movem, espaço atira, P ou Esc pausam.

**Desafio do dia.** Seed pela data → mesma sequência de meteoros e inimigos e mesma meta
(400 a 800 pontos) para todo mundo.

**Naves.** Pioneira (0), Cometa (600), Brasa (1 800), Gelo (4 000), Neon (8 000) e
Aurora (14 000 pontos somados).

## Integração

- Card no hub com o selo "Novo" (saiu do Quebra-Cuca), manifesto próprio em
  `static/astronave/` e ícones gerados de `scripts/icons/astronave{,-maskable}.svg`.
- Token de cor `cosmos` no `tailwind.config.js`.
- Chaves `astronave.*` em pt-BR, en e es.
- `football-e2e/tests/test_astronave.py` — 19 testes: hub, abas, os dois modos de controle
  (arraste, botões, teclado), tamanho da arena em cada modo, pausa, fim de partida com
  persistência, desafio do dia, naves e zerar estatísticas. Entrou no `games.yml`.

## Fora do escopo

- Som e música.
- Chefes de fase e outros tipos de inimigo.
- Movimento vertical da nave.
