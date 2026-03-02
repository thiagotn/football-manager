#!/usr/bin/env bash
# =============================================================================
# setup-vps.sh — Preparação do VPS para o rachao.app
# Testado em: Ubuntu 24.04 LTS
# Uso: sudo bash setup-vps.sh
# =============================================================================
set -euo pipefail

# ── Cores ─────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'

DEPLOY_DIR="/opt/football-manager"

# ── Verificações iniciais ──────────────────────────────────────────────────────
if [[ $EUID -ne 0 ]]; then
  echo -e "${RED}Erro: execute como root.${NC}  sudo bash setup-vps.sh"
  exit 1
fi

if ! grep -q "24.04" /etc/os-release 2>/dev/null; then
  echo -e "${YELLOW}Aviso: script testado no Ubuntu 24.04. Continuando mesmo assim...${NC}"
fi

echo -e "${BOLD}╔══════════════════════════════════════════╗${NC}"
echo -e "${BOLD}║   rachao.app — Setup do VPS              ║${NC}"
echo -e "${BOLD}╚══════════════════════════════════════════╝${NC}\n"

# ── 1. Atualizar o sistema ─────────────────────────────────────────────────────
echo -e "${BLUE}[1/5] Atualizando pacotes do sistema...${NC}"
apt-get update -q
DEBIAN_FRONTEND=noninteractive apt-get upgrade -y -q
apt-get install -y -q curl ca-certificates gnupg ufw openssl

# ── 2. Instalar Docker ─────────────────────────────────────────────────────────
echo -e "${BLUE}[2/5] Instalando Docker...${NC}"
if command -v docker &>/dev/null; then
  echo -e "  ${GREEN}✓ Docker já instalado: $(docker --version)${NC}"
else
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
    | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg

  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu \
$(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
    | tee /etc/apt/sources.list.d/docker.list >/dev/null

  apt-get update -q
  apt-get install -y -q \
    docker-ce docker-ce-cli containerd.io \
    docker-buildx-plugin docker-compose-plugin

  systemctl enable --now docker
  echo -e "  ${GREEN}✓ Docker instalado: $(docker --version)${NC}"
  echo -e "  ${GREEN}✓ Docker Compose: $(docker compose version)${NC}"
fi

# ── 3. Configurar firewall (UFW) ───────────────────────────────────────────────
echo -e "${BLUE}[3/5] Configurando firewall...${NC}"
ufw --force reset >/dev/null
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh        comment 'SSH'
ufw allow 80/tcp     comment 'HTTP  (Traefik)'
ufw allow 443/tcp    comment 'HTTPS (Traefik)'
ufw --force enable >/dev/null
echo -e "  ${GREEN}✓ UFW ativado — regras: SSH, 80/tcp, 443/tcp${NC}"
ufw status numbered

# ── 4. Criar diretório de deploy ───────────────────────────────────────────────
echo -e "\n${BLUE}[4/5] Criando diretório de deploy...${NC}"
mkdir -p "$DEPLOY_DIR"
echo -e "  ${GREEN}✓ $DEPLOY_DIR criado${NC}"

# ── 5. Criar .env.prod ────────────────────────────────────────────────────────
echo -e "${BLUE}[5/5] Configurando variáveis de ambiente...${NC}"
if [[ -f "$DEPLOY_DIR/.env.prod" ]]; then
  echo -e "  ${YELLOW}⚠ .env.prod já existe — mantendo sem alteração.${NC}"
else
  SECRET_KEY=$(openssl rand -hex 32)
  cat > "$DEPLOY_DIR/.env.prod" <<EOF
# ── Banco de Dados ────────────────────────────────────────────────
POSTGRES_DB=football
POSTGRES_USER=postgres
POSTGRES_PASSWORD=TROQUE_POR_UMA_SENHA_FORTE

# ── API ───────────────────────────────────────────────────────────
# Gerado automaticamente pelo setup — pode ser regenerado com: openssl rand -hex 32
SECRET_KEY=${SECRET_KEY}

# ── Traefik / Let's Encrypt ───────────────────────────────────────
# E-mail para notificações de expiração de certificado SSL
ACME_EMAIL=seu@email.com
EOF
  chmod 600 "$DEPLOY_DIR/.env.prod"
  echo -e "  ${GREEN}✓ .env.prod criado em $DEPLOY_DIR/.env.prod${NC}"
  echo -e "  ${YELLOW}  SECRET_KEY gerado automaticamente.${NC}"
  echo -e "  ${YELLOW}  Edite o arquivo para definir POSTGRES_PASSWORD e ACME_EMAIL.${NC}"
fi

# ── Resumo ─────────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}${BOLD}╔══════════════════════════════════════════╗${NC}"
echo -e "${GREEN}${BOLD}║   Setup concluído com sucesso!  ✓        ║${NC}"
echo -e "${GREEN}${BOLD}╚══════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BOLD}Próximos passos:${NC}"
echo ""
echo -e "  ${YELLOW}1.${NC} Edite as variáveis de produção:"
echo -e "     ${BOLD}nano $DEPLOY_DIR/.env.prod${NC}"
echo -e "     Defina: POSTGRES_PASSWORD e ACME_EMAIL"
echo ""
echo -e "  ${YELLOW}2.${NC} Aponte os registros DNS para o IP deste VPS:"
echo -e "     ${BOLD}rachao.app      → $(curl -s ifconfig.me 2>/dev/null || echo '<IP do VPS>')${NC}"
echo -e "     ${BOLD}api.rachao.app  → <mesmo IP>${NC}"
echo -e "     ${BOLD}www.rachao.app  → <mesmo IP>${NC}"
echo ""
echo -e "  ${YELLOW}3.${NC} No GitHub, dispare o deploy:"
echo -e "     ${BOLD}Actions → Deploy to Production → Run workflow${NC}"
echo ""
echo -e "  Versão Docker:  $(docker --version)"
echo -e "  Versão Compose: $(docker compose version)"
echo ""
