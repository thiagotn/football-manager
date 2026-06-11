#!/usr/bin/env bash
# =============================================================================
# setup-vps-ops.sh — Ergonomia de sessão SSH para ops do rachao.app
# Testado em: Ubuntu 24.04 LTS
# Uso: sudo bash setup-vps-ops.sh
#
# O que faz (idempotente — pode rodar várias vezes):
#  - Instala ferramentas de análise/ops (bat, htop, ncdu, jq, tmux, ripgrep,
#    fzf, eza, lazydocker)
#  - Adiciona bloco de aliases / history / prompt em /root/.bashrc
#  - Cria symlink /usr/local/bin/bat → batcat
#  - NÃO substitui binários do sistema, NÃO toca em /etc/bash.bashrc
# Rollback: remova o bloco "rachao.app ops ergonomia" de /root/.bashrc.
# =============================================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'

DEPLOY_DIR="/opt/football-manager"
BASHRC="/root/.bashrc"
BLOCK_MARK="# ── rachao.app ops ergonomia ──"

if [[ $EUID -ne 0 ]]; then
  echo -e "${RED}Erro: execute como root.${NC}  sudo bash setup-vps-ops.sh"
  exit 1
fi

echo -e "${BOLD}╔══════════════════════════════════════════╗${NC}"
echo -e "${BOLD}║   rachao.app — Setup ergonomia ops VPS   ║${NC}"
echo -e "${BOLD}╚══════════════════════════════════════════╝${NC}\n"

# ── 1. Pacotes apt (Tier 1 + Tier 2) ──────────────────────────────────────────
echo -e "${BLUE}[1/4] Instalando pacotes via apt...${NC}"
apt-get update -q
DEBIAN_FRONTEND=noninteractive apt-get install -y -q \
  bat htop ncdu jq tmux git curl \
  ripgrep fzf eza bash-completion

# Symlink bat → batcat (no Debian/Ubuntu o binário é "batcat")
if [[ ! -e /usr/local/bin/bat ]] && command -v batcat &>/dev/null; then
  ln -sf "$(command -v batcat)" /usr/local/bin/bat
  echo -e "  ${GREEN}✓ symlink /usr/local/bin/bat → batcat${NC}"
fi
echo -e "  ${GREEN}✓ pacotes instalados${NC}"

# ── 2. Lazydocker (binário direto) ────────────────────────────────────────────
echo -e "${BLUE}[2/4] Instalando lazydocker...${NC}"
if command -v lazydocker &>/dev/null; then
  echo -e "  ${GREEN}✓ lazydocker já instalado: $(lazydocker --version 2>&1 | head -n1)${NC}"
else
  curl -fsSL https://raw.githubusercontent.com/jesseduffield/lazydocker/master/scripts/install_update_linux.sh \
    | DIR=/usr/local/bin bash >/dev/null
  echo -e "  ${GREEN}✓ lazydocker instalado em /usr/local/bin/lazydocker${NC}"
fi

# ── 3. Bloco de config em /root/.bashrc ───────────────────────────────────────
echo -e "${BLUE}[3/4] Aplicando config em $BASHRC...${NC}"
if grep -qF "$BLOCK_MARK" "$BASHRC" 2>/dev/null; then
  echo -e "  ${YELLOW}⚠ bloco já presente — pulando (re-aplique manualmente removendo o bloco antes).${NC}"
else
  cat >> "$BASHRC" <<'EOF'

# ── rachao.app ops ergonomia ──────────────────────────────────────────────
# Aliases (não substituem binários — só atalhos paralelos)
alias cls='clear'
alias ll='ls -lah --color=auto'
alias la='ls -A --color=auto'
alias l='ls -CF --color=auto'
alias ..='cd ..'
alias ...='cd ../..'
alias grep='grep --color=auto'
alias egrep='egrep --color=auto'
alias bat='batcat'

# Docker / Compose
alias dps='docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"'
alias dlog='docker logs -f --tail=200'
alias dco='docker compose'
alias dcl='docker compose logs -f --tail=200'

# Atalhos rachao.app
alias cdrachao='cd /opt/football-manager'
alias logs-api='docker logs -f --tail=200 football-api'
alias logs-apigo='docker logs -f --tail=200 football-api-go'
alias logs-traefik='docker logs -f --tail=200 football-traefik'
alias logs-front='docker logs -f --tail=200 football-frontend'

# eza substitui `ll` por uma versão com git status e ícones
if command -v eza &>/dev/null; then
  alias ll='eza -lah --git --icons'
  alias tree='eza --tree --level=2'
fi

# Histórico maior, sem duplicatas, append entre sessões
export HISTSIZE=10000
export HISTFILESIZE=20000
export HISTCONTROL=ignoreboth:erasedups
shopt -s histappend
shopt -s checkwinsize

# Prompt: usuário verde | path azul | branch git amarelo | $/# normal
parse_git_branch() {
  git branch 2>/dev/null | sed -n 's/^\* \(.*\)/ (\1)/p'
}
PS1='\[\e[1;32m\]\u@\h\[\e[0m\]:\[\e[1;34m\]\w\[\e[0;33m\]$(parse_git_branch)\[\e[0m\]\$ '

# fzf: Ctrl+R = busca fuzzy no histórico | Ctrl+T = seletor de arquivos
[ -f /usr/share/doc/fzf/examples/key-bindings.bash ] && \
  source /usr/share/doc/fzf/examples/key-bindings.bash
[ -f /usr/share/doc/fzf/examples/completion.bash ] && \
  source /usr/share/doc/fzf/examples/completion.bash

# bash-completion (caso o /etc/bash.bashrc não tenha habilitado)
if ! shopt -oq posix; then
  if [ -f /usr/share/bash-completion/bash_completion ]; then
    . /usr/share/bash-completion/bash_completion
  fi
fi
# ── fim rachao.app ops ────────────────────────────────────────────────────
EOF
  echo -e "  ${GREEN}✓ bloco adicionado em $BASHRC${NC}"
fi

# ── 4. tmux mínimo (opcional, só se ~/.tmux.conf não existir) ─────────────────
echo -e "${BLUE}[4/4] Config mínima do tmux...${NC}"
if [[ -e /root/.tmux.conf ]]; then
  echo -e "  ${YELLOW}⚠ /root/.tmux.conf já existe — mantendo sem alteração.${NC}"
else
  cat > /root/.tmux.conf <<'EOF'
# Prefix Ctrl+a (mais ergonômico que Ctrl+b)
set -g prefix C-a
unbind C-b
bind C-a send-prefix

set -g mouse on
set -g default-terminal "screen-256color"
set -g history-limit 50000

# Status bar simples com hostname + horário
set -g status-bg colour235
set -g status-fg colour250
set -g status-left  '#[fg=green,bold] #H #[default]'
set -g status-right '#[fg=yellow]%Y-%m-%d %H:%M#[default]'

# Split em | e -
bind | split-window -h
bind - split-window -v
EOF
  echo -e "  ${GREEN}✓ /root/.tmux.conf criado${NC}"
fi

# ── Resumo ────────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}${BOLD}╔══════════════════════════════════════════╗${NC}"
echo -e "${GREEN}${BOLD}║   Setup ops concluído!  ✓                ║${NC}"
echo -e "${GREEN}${BOLD}╚══════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BOLD}Para ativar nesta sessão:${NC}  ${YELLOW}source ~/.bashrc${NC}"
echo -e "${BOLD}Ou simplesmente reabra o SSH.${NC}"
echo ""
echo -e "${BOLD}Smoke test:${NC}"
echo -e "  ${YELLOW}cls${NC}                            # limpa a tela"
echo -e "  ${YELLOW}ll${NC}                             # ls colorido com git status (eza)"
echo -e "  ${YELLOW}bat $DEPLOY_DIR/.env.prod${NC}   # syntax highlight"
echo -e "  ${YELLOW}dps${NC}                            # tabela de containers"
echo -e "  ${YELLOW}logs-apigo${NC}                     # logs do api-go ao vivo"
echo -e "  ${YELLOW}lazydocker${NC}                     # TUI de Docker"
echo -e "  ${YELLOW}Ctrl+R${NC}                         # fuzzy search no histórico"
echo ""
echo -e "${BOLD}Rollback:${NC} remover o bloco entre"
echo -e "  ${YELLOW}# ── rachao.app ops ergonomia ──${NC} e ${YELLOW}# ── fim rachao.app ops ──${NC}"
echo -e "de $BASHRC."
echo ""
