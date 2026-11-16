# PRD 058 — Bê-á-Bá (letras e números para crianças)

- **Status:** 🚧 Fase 1 implementada (2026-11-15) · fases 2 e 3 propostas
- **Código:** [`games/src/lib/be-a-ba/`](../../games/src/lib/be-a-ba/) e [`games/src/routes/be-a-ba/`](../../games/src/routes/be-a-ba/) — estado corrente em [`games/CLAUDE.md`](../../games/CLAUDE.md)
- **Onde:** novo jogo no app `games/` → `games.thiagotn.com/be-a-ba/`, mesma stack dos outros (ver [`games/CLAUDE.md`](../../games/CLAUDE.md))
- **Nome provisório:** **Bê-á-Bá** (en *ABC 123*, es *Abecé*). Alternativas: "Letrinhas", "ABC da Turma"

## Problema / Objetivo

Um jogo para crianças em alfabetização (≈ 4 a 7 anos) que estão descobrindo que letras
juntas formam palavras, e que também treine as primeiras operações com números. A cada
objetivo cumprido a criança ganha um prêmio.

Pedido original: mostrar uma imagem e a criança digita o nome correspondente (ex.: uva,
carro, arthur, mesa, elefante), mais alguns tipos de desafio de operações básicas.

### Quem usa

| Pessoa | O que precisa |
|---|---|
| **Criança** | Ainda não lê instruções. Precisa de áudio, botões grandes, nenhum texto obrigatório, nenhuma punição por erro |
| **Adulto (pai/mãe)** | Configurar nível, cadastrar palavras da família (ex.: o nome **Arthur** com foto), ver o progresso |

## Princípios

1. **Ouvir antes de ler.** Toda palavra, letra e número pode ser falado (botão de alto-falante). A instrução de cada tela é falada, não escrita.
2. **Erro não pune.** Sem vidas, sem relógio, sem "errou!" vermelho. A letra errada balança e volta; depois de 2 tentativas o jogo dá uma dica.
3. **Sempre termina bem.** Toda rodada termina com prêmio. A dificuldade se ajusta sozinha (seção *Progressão*).
4. **Rodadas curtas.** 5 desafios por rodada (≈ 2 a 4 min), pensando na atenção da idade.
5. **Privacidade de criança.** Nada sai do aparelho: sem conta, sem anúncio, sem link externo na área da criança, sem analytics. Fotos ficam no próprio aparelho.

## Trilha das Letras

### Modos de jogo (do mais fácil ao mais difícil)

| # | Modo | Como funciona | Fase da escrita |
|---|---|---|---|
| L1 | **Primeira letra** | Imagem + 3 letras grandes. "Com que letra começa **UVA**?" | Pré-silábica → silábica |
| L2 | **Junte as sílabas** | Imagem + sílabas embaralhadas (`CA` `RRO`) para arrastar ou tocar na ordem | Silábica |
| L3 | **Monte a palavra** | Imagem + letras da palavra embaralhadas + 1–2 letras sobrando. A criança toca as letras na ordem | Silábico-alfabética |
| L4 | **Escreva a palavra** | Imagem + **teclado na tela** (é o pedido original: digitar o nome) | Alfabética |
| L5 | **Complete a letra que falta** | `E L E _ A N T E` | Reforço de ortografia |

- **Letra de forma maiúscula (CAIXA ALTA) por padrão**, como a escola começa (BNCC). Os ajustes permitem minúscula depois.
- **Teclado na tela próprio** (não o do sistema): letras grandes em ordem **ABC** (não QWERTY, que a criança não conhece), com `Ç` e botão de apagar. O teclado físico também funciona.
- **Acentos:** a criança digita `AVIAO` e o jogo completa para **AVIÃO**, mostrando o til com uma animação. Acento nunca conta como erro nos níveis iniciais.
- **Letra certa no lugar certo** fica verde e trava. Letra errada balança e volta à bandeja (L3) ou some (L4).
- **Dica progressiva:** 1ª dica fala a palavra devagar, sílaba por sílaba ("e-le-fan-te"); 2ª dica mostra a próxima letra em contorno; 3ª dica preenche a próxima letra.

### Banco de palavras (≈ 120 na v1)

Organizado por estrutura silábica, que é o que define a dificuldade de leitura, não o tamanho da palavra:

| Nível | Estrutura | Exemplos |
|---|---|---|
| 1 | Vogais e palavras de 2–3 letras | UVA, OVO, AVE, RUA, SOL, PÃO, MÃO |
| 2 | 2 sílabas simples (consoante + vogal) | BOLA, CASA, GATO, PATO, MESA, SAPO, DADO, VACA, LUA |
| 3 | 3 sílabas simples | BONECA, SAPATO, MACACO, CAVALO, PANELA, JANELA, TOMATE |
| 4 | Sílabas com dígrafo ou encontro | CARRO, CHAVE, PRATO, FLOR, PEIXE, OLHO, NINHO, PASSARINHO |
| 5 | Palavras longas e mistas | ELEFANTE, BORBOLETA, BICICLETA, GIRAFA, TARTARUGA, ABACAXI |

Cada palavra no banco tem: grafia (com acento), divisão silábica, categoria (fruta, bicho,
casa, transporte, corpo, escola, comida), imagem e nível.

### Palavras da família

Na área dos pais o adulto cadastra palavras próprias, com **foto tirada na hora ou da
galeria**: o nome da criança (**ARTHUR**), MAMÃE, PAPAI, VOVÓ, o nome do cachorro. Elas
entram nas rodadas com prioridade, porque o próprio nome é a palavra que a criança mais
quer escrever.

- Foto redimensionada no aparelho (máx. 512 px) e guardada no **IndexedDB**. Nunca é enviada.
- A divisão silábica de palavras novas é sugerida automaticamente e o adulto pode corrigir (`AR-THUR`).

## Trilha dos Números

Todos os desafios usam **objetos desenhados** junto com o número, para a criança contar se precisar.

| # | Desafio | Exemplo | Faixa |
|---|---|---|---|
| N1 | **Quantos tem?** | 4 maçãs → toque no **4** | 1–10 |
| N2 | **Mostre o número** | "Mostre **7**" → toque nos balões até ter 7 | 1–10 |
| N3 | **Quem tem mais?** | Dois cestos → toque no que tem mais (depois: `>` `<` `=`) | 1–20 |
| N4 | **Qual vem depois?** | `3 4 _ 6` | 1–20, depois de 2 em 2 |
| N5 | **Juntar (soma)** | 🍎🍎 + 🍎🍎🍎 = ? → resposta em 3 opções ou no teclado numérico | até 10, depois até 20 |
| N6 | **Tirar (subtração)** | 5 passarinhos, 2 voam → quantos ficaram? (animação dos 2 saindo) | até 10, depois até 20 |
| N7 | **Número que falta** | `3 + _ = 5` | até 10 |
| N8 | **Dobro e metade** | Espelho que duplica os objetos | até 20 |
| N9 | **Grupos (multiplicação)** | 3 pratos com 2 biscoitos → quantos biscoitos? | tabuadas do 2, 5 e 10 |
| N10 | **Dinheiro de brinquedo** | Moedas de 1, 2, 5 e 10 → pagar o picolé de 7 | até 20 |

- Respostas de 1 dígito: escolha entre 3 opções grandes. A partir de 10, teclado numérico na tela.
- Os números são falados ao tocar ("sete").
- Geração por seed, como nos outros jogos, e sem repetir a mesma conta na mesma rodada.

## Prêmios

| Quando | Prêmio |
|---|---|
| Cada acerto | Estrela voando para o pote + som curto + a palavra falada de novo |
| Rodada completa (5 desafios) | **Figurinha** nova para o álbum (sorteada entre as que faltam) + chuva de confete |
| A cada 5 figurinhas | Página do álbum completa → animação do mascote comemorando |
| Ao subir de nível | Novo **mundo** liberado (Fazenda → Cidade → Praia → Floresta → Espaço), que muda o fundo e as figurinhas disponíveis |
| Primeira vez com o próprio nome | Figurinha especial "Eu sei escrever meu nome!" |

- **Álbum de figurinhas** (≈ 60 na v1, 12 por mundo): é o motivo para voltar. Fica numa aba própria, com as vazias em silhueta.
- **Mascote** com nome, que fala as instruções e comemora (proposta: uma corujinha, "Lili").
- Nada de ranking ou comparação com outras crianças.

## Progressão adaptativa

- Cada palavra e cada tipo de conta tem uma "caixa" de domínio (sistema de **Leitner**, 1 a 5). Acerto sem dica sobe uma caixa; com dica fica; erro desce uma.
- A rodada mistura ≈ 60% conteúdo novo do nível, 30% revisão de caixas baixas e 10% coisa já dominada (para dar confiança).
- Sobe de nível quando ≥ 80% do nível atual está na caixa 3 ou mais. Nunca desce sozinho; o adulto pode ajustar.
- A criança escolhe só a trilha (**Letras** ou **Números**) na tela inicial, com dois botões grandes ilustrados. O resto o jogo decide.

## Área dos pais

Protegida por uma pergunta que criança pequena não resolve sozinha (ex.: "Segure o botão por 3 segundos e toque no número **quarenta e dois**").

- Nome e foto da criança
- Palavras da família (adicionar, editar, remover)
- Trilha: nível de cada uma (automático ou fixo), modos ligados/desligados
- Letras maiúsculas ou minúsculas; voz ligada/desligada e velocidade da fala
- **Relatório:** palavras dominadas, palavras com dificuldade, contas com mais erro, tempo jogado por dia
- Limite de tempo opcional (ex.: 20 min/dia, depois o mascote "vai dormir")
- Mais de uma criança no mesmo aparelho (perfis com foto)

## Imagens e som

| Item | Proposta | Por quê |
|---|---|---|
| Imagens das palavras | **Twemoji** (SVG, licença CC-BY 4.0) num subconjunto de ≈ 150 desenhos, servidos pelo próprio app | Traço consistente em qualquer aparelho, offline, ≈ 1–2 KB cada. Emoji do sistema muda de desenho entre Android e iPhone |
| Palavras sem emoji bom | Desenho próprio em SVG no mesmo estilo | Poucas (ex.: PANELA, JANELA) |
| Voz | **Web Speech API** (`speechSynthesis`, voz pt-BR do aparelho) | Sem custo e funciona offline no Android e iOS. Fallback: sem voz, só texto e animação |
| Efeitos sonoros | 5–6 sons curtos próprios (acerto, letra errada, figurinha, confete) em arquivo pequeno | Precisam ser os mesmos em todo aparelho |

- Créditos da Twemoji na área dos pais (exigência da licença).
- Risco: a qualidade da voz pt-BR varia por aparelho. Plano B na v2: gravar o áudio das ≈ 120 palavras (≈ 1 MB em Opus).

## Arquitetura (igual aos outros jogos)

| Camada | Arquivo |
|---|---|
| Rota | `games/src/routes/be-a-ba/+page.svelte` (tela cheia, `ssr = false`) |
| Lógica pura | `games/src/lib/be-a-ba/letters.ts` (escolha da palavra, checagem letra a letra, acentos, dicas, divisão silábica), `numbers.ts` (gerador seedado por tipo de desafio), `progress.ts` (Leitner, nível, sorteio da rodada) |
| Conteúdo | `words.ts` (banco de palavras), `stickers.ts` (álbum), `worlds.ts` |
| Persistência | `storage.ts` → localStorage `games_be_a_ba` (perfis, progresso, figurinhas, ajustes) + IndexedDB para fotos |
| Voz | `speech.ts` (escolha da voz pt-BR, fila, velocidade, fallback silencioso) |
| Testes | Vitest: checagem de palavra e acentos, divisão silábica do banco inteiro, toda conta gerada com resultado na faixa, Leitner. E2E: `football-e2e/tests/test_be_a_ba.py` |
| Registro | entrada em `games/src/lib/games.ts`, manifesto e ícones em `static/be-a-ba/`, i18n nos 3 idiomas |

**Idioma:** a interface sai em pt-BR, en e es como os outros jogos, mas **o banco de
palavras da v1 é só em português** (alfabetização depende do idioma). Em en/es a trilha
de Letras aparece como "em breve" e a de Números funciona normalmente.

## Fases de entrega

| Fase | Conteúdo | Estimativa |
|---|---|---|
| **1 — MVP** | Tela inicial com as 2 trilhas · Letras L1, L3, L4 com ≈ 60 palavras (níveis 1–3) · Números N1, N3, N5, N6 até 10 · estrelas + álbum com 24 figurinhas · voz · área dos pais com nome da criança e palavras da família | 1 entrega |
| **2** | L2 (sílabas) e L5 · níveis 4–5 do banco (≈ 120 palavras) · N2, N4, N7 · progressão Leitner completa · mundos · relatório para os pais | 1 entrega |
| **3** | N8–N10 · perfis múltiplos · limite de tempo · minúsculas · áudio gravado (se a voz do aparelho não for boa o bastante) | 1 entrega |

## Fora de escopo

- Conta, nuvem, sincronizar entre aparelhos
- Letra cursiva (escrever com o dedo) — ideia para depois da v3
- Reconhecimento de fala (a criança falar a palavra)
- Banco de palavras em inglês e espanhol

## Decisões da fase 1 (2026-11-15)

O plano foi aprovado sem respostas às perguntas abaixo; a fase 1 seguiu as recomendações:

| Pergunta | Decisão |
|---|---|
| Nome | **Bê-á-Bá** (en *ABC 123*, es *Abecé*) |
| Idade | 4 a 7 anos — começa por "Primeira letra" |
| Digitar desde o início? | Não: estágios 1–2 só montam com letras; o teclado (L4) entra no estágio 3 |
| Maiúsculas | Só CAIXA ALTA na fase 1 (minúsculas ficam para a fase 3) |
| Imagens | Twemoji |
| Prêmios | Estrela por acerto, figurinha por rodada, álbum de 24 + figurinha especial do próprio nome, mascote corujinha "Lili" |

### O que a fase 1 entregou

- Letras L1 (primeira letra), L3 (montar) e L4 (escrever), com **74 palavras em 4 níveis** — o nível 4 (CARRO, ELEFANTE…) entrou já na fase 1 porque estava nos exemplos do pedido
- Números N1 (quantos tem), N3 (quem tem mais), N5 (somar) e N6 (tirar), até 10
- 5 estágios nas Letras e 4 nos Números; sobe sozinho depois de 2 rodadas boas seguidas (≥ 4 de 5 sem erro nem dica), nunca desce. Leitner por palavra/tipo de conta já ativo (estava na fase 2)
- Álbum com 24 figurinhas + "Meu nome"; estrelas; subida de fase anunciada no prêmio
- Área dos pais: nome da criança (vira palavra da família automaticamente), palavras da família com foto (IndexedDB), fixar fase, voz e velocidade, relatório (rodadas, estrelas, figurinhas, minutos hoje, palavras que já sabe / para treinar, nível de cada conta), idioma, zerar progresso
- Testes: 19 Vitest (`be-a-ba.test.ts`) e 19 E2E (`football-e2e/tests/test_be_a_ba.py`)

### Ficou para as próximas fases

L2 (sílabas) e L5 (letra que falta) · N2, N4, N7–N10 · mundos · perfis múltiplos · limite de tempo · minúsculas · áudio gravado

## Modo Pequenos — 3 e 4 anos (2026-11-16)

Pedido: um modo também para crianças de 3 e 4 anos. Nessa idade a criança ainda não lê nem
escreve, então o modo troca escrever por **escutar, reconhecer e tocar**:

| Trilha | Desafio | Como funciona |
|---|---|---|
| Letras | **Cadê?** | Ouve "Cadê a uva?" e toca na figura certa (42 palavras concretas, com artigo para a fala) |
| Letras | **Ache a letra igual** | Letra grande no alto, toca na igual. Vogais primeiro; depois B, M, P, S, L, T e a **inicial do nome da criança** |
| Números | **Vamos contar!** | Toca em cada objeto; ele ganha o número e a voz conta "um, dois, três". Não tem erro |
| Números | **Ache o número** | Ouve "Ache o número 3"; numeral grande com bolinhas embaixo |
| Números | **Onde tem?** | "Onde tem 2?" — dois grupos, toca no certo |

- Rodadas de **3** desafios (atenção curta); começa com 2 opções e vai para 3. Estágios: Letras 3, Números 3 (até 3, depois até 5)
- Mesmos prêmios (estrela, figurinha por rodada — sai mais rápido) e a mesma regra de erro sem punição
- **A idade é escolhida na 1ª abertura** ("3–4 anos" ou "5–7 anos") e muda na área dos pais. Os dois modos guardam progresso separado; quem já jogava antes segue no 5–7
- Código: `games/src/lib/be-a-ba/little.ts` e `LittleChallenge.svelte`; 5 testes Vitest e 7 E2E novos

## Perguntas da revisão (respondidas acima)

1. **Nome:** "Bê-á-Bá" está bom, ou prefere outro?
2. **Idade-alvo:** 4 a 7 anos? Muda o ponto de partida (com ou sem o modo "Primeira letra").
3. **Digitar desde o início?** Proponho começar por montar com letras embaralhadas (L3) e liberar o teclado (L4) depois. Ou prefere o teclado já no primeiro nível, como no pedido?
4. **Maiúsculas:** começar em CAIXA ALTA (recomendado) e liberar minúsculas nos ajustes?
5. **Imagens:** Twemoji serve, ou prefere um estilo ilustrado próprio (mais bonito, mais trabalho)?
6. **Mascote e prêmios:** álbum de figurinhas + mascote coruja está bom? Outra ideia de prêmio que funcione com a criança?
7. **Fases:** começo pela Fase 1 (MVP) como descrita?
