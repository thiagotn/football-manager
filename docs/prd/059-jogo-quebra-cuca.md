# PRD 059 — Quebra-Cuca (quebra-cabeça infantil)

- **Status:** ✅ Implementado (2026-11-20)
- **Código:** [`games/src/lib/quebra-cuca/`](../../games/src/lib/quebra-cuca/) e [`games/src/routes/quebra-cuca/`](../../games/src/routes/quebra-cuca/) — estado corrente em [`games/CLAUDE.md`](../../games/CLAUDE.md)
- **Onde:** `games.thiagotn.com/quebra-cuca/`, mesma stack dos outros jogos
- **Nome:** **Quebra-Cuca** (en *Puzzle Pals*, es *Rompecocos*)

## Problema / Objetivo

Pedido: "um jogo para criança, de quebra-cabeça, com tema de dinossauro, heróis, fazenda,
circo, onde a criança arraste e solte as peças nos lugares para ir montando".

Público: 3 a 7 anos, no celular ou tablet dos pais. A criança não lê: tudo precisa funcionar
só com figura, toque e arrastar.

## Princípios

1. **Peça de quebra-cabeça de verdade.** Orelhas e buracos, não quadradinhos — é o que a criança reconhece.
2. **Erro não existe.** Soltar no lugar errado só faz a peça voltar para a bandeja. Sem relógio, sem pontos perdidos.
3. **Figura de fundo como guia.** A cena aparece clarinha no tabuleiro (pode ser desligada nos ajustes para quem já é maior).
4. **Começa fácil.** 4 peças (2×2) para os pequenos; 6, 9 e 12 para os maiores. O jogo lembra a última escolha.
5. **Offline e privado.** Sem conta, sem anúncio, sem rede depois de instalado.

## Conteúdo

4 temas × 3 cenas, montadas com desenhos Twemoji (CC-BY 4.0) sobre céu e morros desenhados no SVG:

| Tema | Cenas |
|---|---|
| Dinossauros | Vale dos dinossauros · O ninho · Dinossauros à noite |
| Heróis | Super-herói · Heroína no espaço · Heróis do dia a dia (bombeiro, ambulância) |
| Fazenda | O sítio · Bichos da fazenda · A horta |
| Circo | A lona do circo · O picadeiro · Parque de diversões |

Heróis são **genéricos** (capa e máscara, profissões) — nenhum personagem de marca.

## Fluxo

Início (4 temas, progresso `x/3`) → cenas do tema (estrelas) → "Quantas peças?" (4/6/9/12) →
montar → "Parabéns!" com confete e voz → **Próximo** (próxima cena do tema), **mais peças**
(mesma cena, próximo tamanho) ou **Escolher outro**.

Estrelas por cena = maior tamanho já montado (4 → 1★ … 12 → 4★).

## Mecânica

- Encaixes sorteados por partida (seed); vizinhos sempre complementares, borda reta.
- A peça é um recorte (`clipPath`) da cena; a bandeja fica embaixo do tabuleiro e encolhe as peças se não couberem.
- Arrastar com o dedo (pointer events); encaixa quando o centro solto está a até 40% da célula do lugar certo.
- Voz (`speechSynthesis`) anuncia a cena e comemora no fim; desliga nos ajustes.

## Fora do escopo (ideias futuras)

- Girar peças; peças com formatos irregulares; foto da família como quebra-cabeça.
- Mais temas (fundo do mar, espaço, veículos).
