# PRD — Simulador Público de Sorteio de Times
## Rachao.app · Página pública para testar montagem de times por posição e estrelas

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | 📋 Proposto |
| **Data** | Abril de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Contexto e Motivação

O rachao.app já possui o algoritmo de sorteio de times integrado ao fluxo de partidas reais (PRD 012), mas esse fluxo exige login, vínculo com grupo e confirmação de presença.

Existe a necessidade de uma **ferramenta pública e independente** para que qualquer pessoa possa:
- Testar o algoritmo de sorteio sem precisar de conta ou grupo cadastrado.
- Simular cenários com listas de jogadores reais do seu rachão, atribuindo estrelas e posições.
- Compartilhar o resultado do sorteio facilmente (ex: via WhatsApp).

Esta página também serve como **vitrine da funcionalidade** para novos usuários que ainda não criaram uma conta, evidenciando o valor do rachao.app antes do cadastro.

A diferença central em relação ao PRD 012 é que o simulador considera **múltiplas posições** (não apenas goleiro/linha), distribuindo as posições proporcionalmente entre os times para gerar escalações mais realistas.

---

## 2. Requisitos Funcionais

### RF-01 — Rota pública sem autenticação

A página está disponível em `/draw` (inglês, conforme padrão de rotas do projeto).

- Acessível por qualquer visitante sem login.
- Não há dados persistidos no servidor — todo o estado é gerenciado no cliente (localStorage para preservar edições entre reloads, sem backend).
- A página não aparece na navegação principal nem no menu de usuário autenticado (acesso direto por URL ou link compartilhado).

---

### RF-02 — Lista de jogadores padrão (seed de 30 jogadores)

Ao acessar `/draw` pela primeira vez, a página exibe automaticamente uma lista de **30 jogadores pré-gerados** com dados fictícios.

Cada jogador possui:

| Campo | Tipo | Descrição |
|---|---|---|
| `name` | string | Nome gerado (ex: "Carlos Silva") |
| `nickname` | string | Apelido gerado (ex: "Carlão") |
| `stars` | int 1–5 | Nota de habilidade — distribuída de forma realista (ver seção 5) |
| `position` | enum | Posição em campo (ver RF-03) |
| `active` | bool | Se o jogador está na lista ativa para o sorteio (padrão: `true`) |

Os dados de seed são definidos estaticamente no código do frontend. O usuário pode editar qualquer campo antes de sortear.

---

### RF-03 — Posições disponíveis

O simulador expande o modelo do PRD 012 para incluir 5 posições:

| Código | Nome PT | Nome EN | Abreviação |
|---|---|---|---|
| `goalkeeper` | Goleiro | Goalkeeper | GK |
| `defender` | Zagueiro | Defender | ZAG |
| `fullback` | Lateral | Fullback | LAT |
| `midfielder` | Meio-campo | Midfielder | MEI |
| `forward` | Atacante | Forward | ATA |

O seletor de posição usa um dropdown compacto (ou botões de pill) com as abreviações.

---

### RF-04 — Edição inline de jogadores

Cada linha da lista de jogadores permite edição direta, sem modal:

- **Nome/Apelido**: campo de texto editável inline (clique para editar).
- **Estrelas**: seletor visual de 1 a 5 estrelas (clique na estrela desejada).
- **Posição**: dropdown com as 5 posições.
- **Ativar/Desativar**: toggle ou checkbox para incluir/excluir o jogador do próximo sorteio sem removê-lo da lista. Jogadores desativados aparecem com opacidade reduzida.
- **Remover**: botão de remoção (ícone de lixeira) com confirmação inline simples (ex: highlight vermelho no botão ao confirmar).

---

### RF-05 — Adicionar jogador

Botão **"+ Adicionar jogador"** no rodapé da lista inclui um novo jogador com campos em branco (apelido obrigatório antes de sortear). O jogador é adicionado ao final da lista como ativo.

---

### RF-06 — Configuração do sorteio

Acima do botão de sortear, o usuário configura:

| Campo | Tipo | Padrão | Descrição |
|---|---|---|---|
| Jogadores por time | int (2–11) | 5 | Número de jogadores de **linha** por time (excluindo goleiro). Time real = este valor + 1. |
| Número de times | int (2–8) | Calculado automaticamente | Calculado como `floor(ativos / (jogadores_por_time + 1))`, mas editável manualmente. |

> Quando o número de times é calculado automaticamente, exibir a fórmula de forma legível: *"Com 30 jogadores e times de 6, serão 5 times (30 jogadores, 0 reservas)"*.

---

### RF-07 — Validações antes do sorteio

Antes de executar o sorteio, validar:

| Condição | Mensagem de erro |
|---|---|
| Menos de `(jogadores_por_time + 1) * 2` jogadores ativos | "Você precisa de pelo menos X jogadores ativos para 2 times." |
| Nenhum goleiro ativo na lista | Aviso (não bloqueante): "Nenhum goleiro na lista — o sorteio será feito sem garantia de goleiro por time." |
| Jogador sem apelido/nome | "Todos os jogadores precisam ter pelo menos um nome ou apelido." |

---

### RF-08 — Algoritmo de sorteio com posições múltiplas

O algoritmo roda inteiramente no frontend (JavaScript/TypeScript). Não há chamada à API.

**Inputs:**
- Lista de jogadores ativos com `stars` e `position`.
- `players_per_team` (jogadores de linha, excluindo GK).
- `n_times`.

**Lógica:**

```
team_size = players_per_team + 1
n_times = configurado pelo usuário (ou calculado)

1. Separar goleiros dos demais jogadores ativos.

2. Distribuir 1 goleiro por time (snake entre times, ordenados por stars desc):
   - Se há ≥ n_times goleiros: 1 por time, excedentes vão ao pool geral.
   - Se há < n_times goleiros: os times sem goleiro recebem o melhor jogador disponível
     do pool como "goleiro improvisado" (flag visual na página de resultado).

3. Separar o restante em grupos por posição:
   - defenders  (zagueiro)
   - fullbacks   (lateral)
   - midfielders (meio)
   - forwards    (atacante)

4. Para cada grupo de posição, aplicar snake draft entre os times:
   - Ordenar jogadores do grupo por stars desc.
   - Snake: time 1 → time 2 → ... → time N → time N → ... → time 1, repetindo.
   - Continuar até esgotar o grupo.

5. Ao final, se algum time ainda não atingiu team_size, preencher com jogadores
   remanescentes de qualquer posição (snake por stars, sem critério posicional).

6. Jogadores que ultrapassam n_times * team_size são reservas (mantidos em lista separada).

Objetivo: cada time deve ter soma de stars a mais próxima possível dos demais
e distribuição posicional proporcional.
```

**Critério de desempate no snake:** quando dois jogadores têm mesma nota, a ordem é aleatória (embaralhar antes de ordenar por stars).

---

### RF-09 — Exibição do resultado do sorteio

Após clicar em **"Sortear times"**, os times gerados são exibidos abaixo da lista, substituindo qualquer resultado anterior.

**Layout dos times:**
- Cards em grid: 1 coluna em mobile, 2 colunas em tablet, 3+ em desktop.
- Cada card exibe:
  - **Nome do time** (gerado automaticamente — reusa o mesmo banco de nomes do PRD 012, com novos nomes para totalizarem ≥ 40 opções).
  - **Soma de estrelas** do time (ex: `★ 18`).
  - **Lista de jogadores**: apelido (ou nome), ícone/badge da posição (ex: `GK`, `ZAG`), estrelas.
  - **Indicador visual** caso o time não tenha goleiro confirmado (badge amarelo "sem GK").
- **Reservas**: seção separada abaixo dos times, com título "Reservas" e lista de jogadores.

**Equilíbrio visual entre times:**
- Exibir, ao final do resultado, uma linha de rodapé mostrando a soma de estrelas de cada time lado a lado (ex: `Time A: ★18 · Time B: ★17 · Time C: ★18`), evidenciando o balanceamento.

---

### RF-10 — Remontar times

Botão **"Remontar"** disponível após o primeiro sorteio. Executa novo sorteio com os mesmos jogadores e configurações atuais (sem confirmação — não é destrutivo, pois não há dados no servidor). A aleatoriedade garante resultado diferente a cada chamada.

---

### RF-11 — Compartilhar resultado

Botão **"Compartilhar"** (ícone de compartilhamento) disponível após o sorteio.

- Comportamento: copia a URL atual para o clipboard.
- A URL não encoda os times gerados (o sorteio não é persistido) — o destinatário verá a lista de jogadores, mas precisará sortear novamente.
- Toast de confirmação: *"Link copiado!"*

> **Fora de escopo v1:** persistência do resultado via URL (hash ou backend). Avaliar em v2 se houver demanda.

---

### RF-12 — Persistência local (localStorage)

A lista de jogadores editada pelo usuário é salva automaticamente no `localStorage` da chave `draw_players`. Ao recarregar a página, a lista salva é restaurada.

Botão **"Restaurar padrão"** (discreto, no rodapé ou header) apaga o localStorage e recarrega os 30 jogadores de seed originais, com confirmação inline.

---

### RF-13 — Internacionalização (i18n)

Todos os textos visíveis devem usar `$t('draw.*')` conforme padrão do projeto (Paraglide). Adicionar chaves em `pt-BR.json`, `en.json` e `es.json`.

---

## 3. Requisitos Não-Funcionais

- **Mobile-first**: a página deve ser totalmente utilizável em telas de 375px de largura. A lista de jogadores é o elemento central — cada linha deve ser compacta e toques precisos.
- **Sem backend**: nenhuma chamada à API. Todo o estado é local. Isso garante que a página funcione mesmo sem conta e sem impacto nos servidores.
- **Performance**: o algoritmo de sorteio deve rodar em < 50ms mesmo com 50 jogadores (JavaScript síncrono, sem Workers necessário).
- **SSR/SEO**: a rota pode usar `export const ssr = false` já que é uma ferramenta interativa sem necessidade de indexação. Meta tags básicas (`title`, `description`) devem estar presentes para o caso de o link ser compartilhado.

---

## 4. UX — Wireframe de Referência

```
┌─────────────────────────────────────────────────┐
│  ⚽  Simulador de Sorteio                        │
│  Monte e teste seus times antes do rachão       │
│                              [Restaurar padrão] │
├─────────────────────────────────────────────────┤
│  Jogadores (28 ativos de 30)         [+ Adicionar]│
│                                                 │
│  ┌──────────────────────────────────────────┐   │
│  │ ✓  Carlão        ★★★★☆  [MEI]  [🗑]     │   │
│  │ ✓  Pedrinho      ★★★☆☆  [ATA]  [🗑]     │   │
│  │ ✓  Zé Goleiro    ★★★★★  [GK ]  [🗑]     │   │
│  │ –  Marquinhos    ★★☆☆☆  [ZAG]  [🗑]     │   │  ← desativado
│  │ ...                                      │   │
│  └──────────────────────────────────────────┘   │
│                                                 │
│  ─────────────────────────────────────────────  │
│  Configuração do sorteio                        │
│  Jogadores por time: [5 ▼]   Times: [5] (auto) │
│  Com 28 ativos e times de 6: 4 times, 4 reservas│
│                                                 │
│            [ ⚡ Sortear times ]                  │
│                                                 │
│ ─── Resultado ───────────────────────────────── │
│                                                 │
│  ┌──────────────┐  ┌──────────────┐             │
│  │ Leões do     │  │ Barsemlona   │             │
│  │ Asfalto  ★18 │  │          ★17│             │
│  │ GK Zé ★★★★★ │  │ GK João ★★★ │             │
│  │ ZAG Marcos★★★│  │ ZAG Tião ★★ │             │
│  │ MEI Carlão★★★│  │ MEI Pedro ★★│             │
│  │ ATA Caio ★★★│  │ ATA Léo  ★★★│             │
│  └──────────────┘  └──────────────┘             │
│                                                 │
│  ★ Equilíbrio: Time A: 18 · Time B: 17          │
│                                                 │
│  Reservas: Marquinhos (ZAG ★★)                  │
│                                                 │
│  [🔁 Remontar]              [📋 Compartilhar]   │
└─────────────────────────────────────────────────┘
```

---

## 5. Seed de Jogadores Padrão

Os 30 jogadores de seed devem ter distribuição realista de posições e estrelas:

| Posição | Quantidade | Justificativa |
|---|---|---|
| Goleiro (GK) | 4 | ~1 por time em jogo de 5x5 típico |
| Zagueiro (ZAG) | 6 | 2 por time |
| Lateral (LAT) | 4 | 1–2 por time |
| Meio-campo (MEI) | 8 | Posição mais numerosa |
| Atacante (ATA) | 8 | Empatado com MEI |

Distribuição de estrelas (bell curve leve):

| Estrelas | Quantidade |
|---|---|
| ★☆☆☆☆ (1) | 2 |
| ★★☆☆☆ (2) | 6 |
| ★★★☆☆ (3) | 12 |
| ★★★★☆ (4) | 7 |
| ★★★★★ (5) | 3 |

Nomes e apelidos em estilo "várzea brasileira" (ex: Carlão, Tião, Zé Grilo, Ferreirinha, Pintado, Burrinho, Gauchinho, Marcelão, Dedé, Fininho etc.).

---

## 6. Banco de Nomes de Times (Ampliado)

Reutilizar os nomes do PRD 012 e adicionar novos para totalizar ≥ 40 opções:

**Novos nomes a adicionar:**
```
Dínamo de Boteco, Leões do Asfalto, Tubarões da Várzea,
Garotos do Fundão, Titãs do Campo Sujo, Unidos do Barro,
Forja FC, Dragões da Periferia, Guerreiros do Baldão,
Estrelas do Zé, Cansados do Joelho, Cruzeiro do Bairro,
Porto Suado, Benficado, Raio que o Parta FC,
Seleção do Cervejinho, Os Pesados, Amigos do Couro,
Galera do Baldão, Nacional do Piscinão, Herói do Banco
```

---

## 7. Estrutura de Arquivos (Frontend Only)

| Arquivo | Tipo de mudança |
|---|---|
| `src/routes/draw/+page.svelte` | Novo — página principal do simulador |
| `src/lib/utils/team-builder.ts` | Novo — algoritmo de sorteio (TypeScript puro, reutilizável) |
| `src/lib/data/draw-seed.ts` | Novo — dados dos 30 jogadores padrão |
| `src/lib/data/team-names.ts` | Novo (ou extensão) — banco de nomes de times |
| `messages/pt-BR.json` | Chaves `draw.*` |
| `messages/en.json` | Chaves `draw.*` |
| `messages/es.json` | Chaves `draw.*` |

> **Sem alterações no backend.** Nenhuma migration, nenhum endpoint novo.

---

## 8. Critérios de Aceitação

- [ ] `/draw` é acessível sem autenticação, sem redirecionamento para login.
- [ ] Página carrega com 30 jogadores pré-gerados (seed) ao ser acessada pela primeira vez.
- [ ] Todos os campos de jogador (nome, apelido, estrelas, posição) são editáveis inline.
- [ ] Jogadores podem ser ativados/desativados e removidos da lista.
- [ ] Novo jogador pode ser adicionado via botão "+ Adicionar jogador".
- [ ] Configuração de "jogadores por time" está disponível e o número de times é calculado automaticamente (com opção de sobrescrever).
- [ ] Sorteio respeita a regra de 1 goleiro por time quando há goleiros suficientes.
- [ ] Em caso de goleiros insuficientes, o time sem GK exibe indicador visual de aviso.
- [ ] Jogadores são distribuídos por posição e estrelas via snake draft.
- [ ] Soma de estrelas de cada time é exibida e está equilibrada (diferença ≤ 2 estrelas entre times em cenário ideal com 30 jogadores).
- [ ] Reservas são listadas separadamente quando `total_ativos % team_size != 0`.
- [ ] Botão "Remontar" gera novo sorteio com aleatoriedade diferente.
- [ ] Botão "Restaurar padrão" recarrega os 30 jogadores originais após confirmação.
- [ ] Lista editada é persistida no `localStorage` e restaurada ao recarregar a página.
- [ ] Botão "Compartilhar" copia a URL para o clipboard com feedback toast.
- [ ] Todos os textos usam chaves i18n (`$t('draw.*')`).
- [ ] Página é utilizável em mobile (375px), sem overflow horizontal.
- [ ] Validações bloqueiam o sorteio quando condições mínimas não são atendidas, com mensagens claras.

---

## 9. Fora de Escopo (v1)

- Persistência dos times gerados no servidor (sem URL compartilhável com resultado embutido).
- Histórico de sorteios anteriores.
- Edição manual de times após o sorteio (arrastar/dropar jogadores entre times).
- Importação de jogadores de um grupo real do usuário logado.
- Definição de número máximo de zagueiros, laterais etc. por time.
- Foto/avatar de jogador no simulador.
- Modo torneio (bracket de confrontos).

---

## 10. Evolução Futura (v2 — não comprometido)

- **Importar do grupo**: usuário logado pode carregar a lista de confirmados de uma partida real para o simulador.
- **URL com estado**: codificar jogadores e resultado em query params ou hash para resultado compartilhável.
- **Configuração avançada de posições**: definir quantos jogadores de cada posição são esperados por time, com validação antes do sorteio.
- **CTA de conversão**: ao final do resultado, exibir card "Gostou? Crie seu grupo no rachao.app grátis" para converter visitantes em usuários.

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
