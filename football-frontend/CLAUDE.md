# Frontend — Estado atual

> Este arquivo documenta o **estado corrente** do frontend: rotas, componentes e stores existentes.
> Deve ser atualizado sempre que uma nova rota, componente ou store for criado.
> Padrões de implementação (page header, i18n, Svelte 5) estão em `CLAUDE.md` na raiz do projeto.

---

## Rotas existentes (`src/routes/`)

### Públicas (sem autenticação)
| Rota | Descrição |
|------|-----------|
| `/` | Redirect para `/dashboard` ou `/login` |
| `/lp` | Landing page unificada (organizadores + jogadores) |
| `/faq` | FAQ público |
| `/plans` | Comparativo de planos |
| `/privacy` | Política de privacidade |
| `/terms` | Termos de uso |
| `/login` | Login + esqueci minha senha |
| `/register` | Cadastro (3 steps: WhatsApp → OTP → form) |
| `/invite/[token]` | Entrada via convite |
| `/match/[hash]` | Detalhe da partida (público + autenticado) |
| `/match/[hash]/teams` | Sorteio de times (público) |
| `/match/[hash]/results` | Resultado da votação com pódio (público) |
| `/ranking` | Ranking geral da plataforma — top 10 melhores e decepções (público) |
| `/simulator` | Simulador público de sorteio de times — sem login, sem backend, estado em localStorage |
| `/draw` | Redirect 301 para `/simulator` (compatibilidade com URLs salvas) |
| `/discover` | Rachões públicos abertos com filtros (público + autenticado) |
| `/players/[id]` | Perfil público do jogador com Rachão Score |

### Autenticadas
| Rota | Descrição |
|------|-----------|
| `/dashboard` | Home do jogador (redireciona admin para `/admin`) |
| `/groups` | Listagem de grupos do usuário |
| `/groups/new` | Criação de grupo |
| `/groups/[id]` | Detalhe do grupo (abas: Próximos / Últimos / Jogadores / Estatísticas / Financeiro) |
| `/profile` | Perfil, troca de senha, upload/remoção de avatar |
| `/profile/stats` | Estatísticas pessoais |
| `/account/subscription` | Plano atual + upgrade |
| `/account/mcp-tokens` | Tokens MCP pessoais — criação, listagem e revogação |
| `/chat` | Assistente IA — visível e acessível apenas para players com `chat_enabled = true` |
| `/account/checkout` | Retorno do checkout Stripe (success/failure) |
| `/review` | Avaliação do app |
| `/players` | Listagem de jogadores (admin do grupo) |
| `/matches/[slug]` | (legado) |

### Admin (role = 'admin')
| Rota | Descrição |
|------|-----------|
| `/admin` | Painel super admin: big numbers + cadastros |
| `/admin/faq` | Gestão do FAQ |
| `/admin/groups` | Listagem global de grupos |
| `/admin/matches` | Listagem global de rachões |
| `/admin/players` | Gestão de jogadores |
| `/admin/reviews` | Avaliações do app |
| `/admin/subscriptions` | Gestão de assinaturas |
| `/admin/chat` | Gestão de acesso ao Assistente IA — habilitar/desabilitar por usuário |

---

## Componentes disponíveis (`src/lib/components/`)

| Componente | Uso |
|------------|-----|
| `AvatarImage.svelte` | Avatar do jogador: foto ou iniciais com cor determinística. Props: `name`, `avatarUrl?`, `updatedAt?`, `size?` (default 40), `class?` |
| `ConfirmDialog.svelte` | Confirmações destrutivas — bottom sheet mobile / modal desktop. Props: `bind:open`, `message`, `confirmLabel`, `danger`, `onConfirm` |
| `MatchBannerCard.svelte` | Banner do card de partida (campo + logo + dados). Props: `match`, `isGroupAdmin?`, `togglingStatus?`, `onToggleOpen?`, `onAskClose?`. Aceita `children` (slot) para conteúdo extra dentro do card (ex: scoreboard). Usado em `/match/[hash]` e `/match/[hash]/teams`. |
| `DatePicker.svelte` | Seletor de data |
| `LanguageSwitcher.svelte` | Seletor de idioma (pt-BR / en / es) |
| `Modal.svelte` | Modal genérico |
| `Navbar.svelte` | Barra de navegação principal |
| `PageBackground.svelte` | Wrapper obrigatório de fundo para todas as páginas |
| `PhoneInput.svelte` | Input de telefone com seletor de país (26 países) e validação E.164. Usar em todos os formulários com número WhatsApp |
| `PwaInstallButton.svelte` | Botão de instalação PWA |
| `PwaSmartBanner.svelte` | Banner de instalação PWA |
| `PositionSelector.svelte` | Seletor de posição (gk/zag/lat/mei/ata) com badges coloridos. Props: `bind:value` (API value), `readonly?` |
| `StarRating.svelte` | Seletor de estrelas (1–5), usado para `skill_stars` |
| `TimePicker.svelte` | Seletor de horário |
| `Toast.svelte` | Notificações toast |
| `UpsellModal.svelte` | Modal de upgrade de plano |
| `VoteForm.svelte` | Formulário de votação pós-partida (Top 5 + Decepção) |
| `VoteResults.svelte` | Exibição do pódio e ranking de votação |
| `WaitlistModal.svelte` | Modal de candidatura à lista de espera |
| `WaitlistPanel.svelte` | Painel admin para aprovar/rejeitar candidatos |
| `MCPTokenCreateModal.svelte` | Modal dois passos: form de criação de token MCP + exibição única do token gerado. Props: `bind:open`, `onCreated`, `onClose` |

---

## Stores (`src/lib/stores/`)

| Store | Responsabilidade |
|-------|-----------------|
| `auth.ts` | `authStore`, `isAdmin`, `currentPlayer`, `isChatEnabled` — estado de autenticação global |
| `pwaInstall.ts` | Evento `beforeinstallprompt` para botão de instalação PWA |
| `sessionExpired.ts` | Flag para exibir modal de sessão expirada |
| `theme.ts` | Tema claro/escuro |
| `toast.ts` | Fila de toasts (`showToast`, `dismissToast`) |

---

## Libs utilitárias (`src/lib/`)

| Arquivo | Uso |
|---------|-----|
| `team-builder.ts` | Algoritmo de sorteio de times (TypeScript puro, sem API). Tipos: `DrawPlayer`, `Team`, `TeamResult`. Constantes: `POS_ABBR`, `POS_COLOR_CLASSES`, `TEAM_COLORS`. |
| `draw-seed.ts` | 30 jogadores de seed para o simulador `/draw`. Exporta `seedWithIds()`. |
| `team-names.ts` | Banco de ≥ 40 nomes de times estilo várzea. Exporta `TEAM_NAMES` e `shuffledNames()`. |

---

## Namespaces de `api.ts`

`auth` · `players` (inclui `getPublicStats`) · `groups` · `matches` · `push` · `subscriptions` · `votes` · `reviews` · `admin` · `teams` · `finance` · `invites` · `ranking` · `mcpTokens` · `chat`

---

## Rebuild e ambiente local

```bash
# Rebuild completo do frontend (necessário para mudanças JS/Svelte)
cd /home/thiagotn/Documentos/Dev/Projects/football-manager/football-api
sudo docker compose up --build --no-cache

# Limpar localStorage do browser ao testar com nova conta
```
