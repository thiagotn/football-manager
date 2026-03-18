# PRD — Observabilidade (rachao.app)

**Status:** Em revisão
**Data:** 2026-03-17
**Contexto:** VPS único (Hostinger KVM1, 1 vCPU, 4 GB RAM, 50 GB disco, Ubuntu 24.04 LTS), Docker Compose + Traefik.

---

## 1. Problema

Hoje não há visibilidade sobre o comportamento da plataforma em produção:

- Falhas na API são detectadas apenas quando um usuário reporta
- Não há histórico de uso de CPU, RAM ou disco — risco de o servidor ficar sem recurso sem aviso
- Logs existem (structlog no FastAPI, access logs do Traefik) mas só são acessíveis via `docker logs` no terminal, sem pesquisa nem persistência estruturada
- Não há alertas automáticos para indisponibilidade

---

## 2. O que já existe a favor

| Item | Detalhe |
|---|---|
| **structlog** no FastAPI | Logs já em JSON estruturado (`level`, `event`, timestamps) |
| **Traefik** | Access logs com status HTTP, latência e rota |
| **Docker Compose** | Logs de todos os containers disponíveis via socket Docker |
| **Traefik TLS** | Qualquer novo subdomínio recebe HTTPS automático via Let's Encrypt |

---

## 3. Restrições

- Solução **100% open-source** (sem custo de SaaS)
- Rodar no mesmo VPS da aplicação (sem servidor separado)
- Não comprometer a estabilidade da aplicação principal

---

## 4. Proposta 1 — Mínima

### Objetivo
Detectar indisponibilidade e visualizar logs em tempo real com impacto mínimo nos recursos.

### Ferramentas

| Ferramenta | Função | RAM estimada |
|---|---|---|
| **Uptime Kuma** | Monitora endpoints e envia alertas (Telegram, e-mail, WhatsApp) | ~150 MB |
| **Dozzle** | Visualizador de logs Docker em tempo real via web UI | ~10 MB |
| **Total adicional** | | **~160 MB** |

### O que resolve
- Alerta imediato quando `rachao.app`, `api.rachao.app` ou qualquer endpoint cair
- Visualização de logs de todos os containers sem precisar de terminal
- Histórico de uptime com gráficos de disponibilidade

### O que não resolve
- Sem métricas de CPU/RAM/disco históricas
- Logs não são persistidos nem pesquisáveis (Dozzle é somente tempo real)
- Sem visibilidade de erros na API por rota ou latência

### Acesso
| URL | Autenticação |
|---|---|
| `uptime.rachao.app` | Login nativo do Uptime Kuma |
| `logs.rachao.app` | Traefik Basic Auth (htpasswd) |

---

## 5. Proposta 2 — Intermediária ✅ Recomendada

### Objetivo
Métricas históricas de infra + métricas da API + alertas de disponibilidade. Sem persistência de logs.

### Ferramentas

| Ferramenta | Função | RAM estimada |
|---|---|---|
| **Prometheus** | Coleta e armazena métricas (scrape de cAdvisor, Node Exporter e API) | ~120 MB |
| **Grafana** | Dashboards sobre métricas do Prometheus | ~150 MB |
| **cAdvisor** | Métricas dos containers Docker (CPU, RAM, rede por container) | ~60 MB |
| **Node Exporter** | Métricas do host (CPU, RAM, disco, rede) | ~20 MB |
| **Uptime Kuma** | Alertas de disponibilidade | ~150 MB |
| **Total adicional** | | **~500 MB** |

### Impacto no VPS
| Recurso | Atual | Com Proposta 2 | Sobra |
|---|---|---|---|
| RAM | 840 MB (21%) | ~1.350 MB (34%) | ~2.650 MB |
| Disco | 5 GB (10%) | ~7 GB* (14%) | ~43 GB |
| CPU | 1% | +2–5% durante scrapes | Ainda folgado |

*Prometheus com retenção configurada para 15 dias e limite de 2 GB.

### O que resolve
- Histórico de CPU, RAM, disco e rede do host e por container
- Métricas da API: requisições por rota, latência (p50/p95/p99), taxa de erros 4xx/5xx
- Dashboards prontos no Grafana (importar IDs públicos do Grafana.com)
- Alertas quando CPU > 80%, RAM > 85%, disco > 80%
- Alertas de endpoint fora do ar (Uptime Kuma)

### Instrumentação necessária na API
Adicionar `prometheus-fastapi-instrumentator` ao FastAPI — 3 linhas de código, expõe `/metrics` automaticamente. O Prometheus passa a coletar:
- `http_requests_total` — total de requests por rota e status
- `http_request_duration_seconds` — histograma de latência
- `http_requests_in_progress` — requests em andamento

### O que não resolve
- Logs não são persistidos nem pesquisáveis centralmente
- Para correlacionar um pico de erro com o log correspondente, ainda seria preciso `docker logs` no terminal

### Acesso
| URL | Autenticação |
|---|---|
| `grafana.rachao.app` | Login nativo do Grafana |
| `uptime.rachao.app` | Login nativo do Uptime Kuma |
| `prometheus.rachao.app` | Traefik Basic Auth |
| `cadvisor.rachao.app` | Traefik Basic Auth (ou só interno) |

---

## 6. Proposta 3 — Completa (PLG Stack)

### Objetivo
Tudo da Proposta 2 + logs estruturados persistidos, indexados e pesquisáveis. Correlação entre métricas e logs no mesmo painel.

### Ferramentas

| Ferramenta | Função | RAM estimada |
|---|---|---|
| Tudo da Proposta 2 | (ver acima) | ~500 MB |
| **Loki** | Armazena e indexa logs (como "Prometheus para logs") | ~200 MB |
| **Promtail** | Coleta logs dos containers Docker e envia ao Loki | ~50 MB |
| **Total adicional** | | **~750 MB** |

### Impacto no VPS
| Recurso | Atual | Com Proposta 3 | Sobra |
|---|---|---|---|
| RAM | 840 MB (21%) | ~1.590 MB (40%) | ~2.410 MB |
| Disco | 5 GB (10%) | ~9 GB* (18%) | ~41 GB |

*Loki com retenção configurada para 30 dias.

### O que resolve (além da Proposta 2)
- Todos os logs de todos os containers indexados e pesquisáveis via LogQL no Grafana
- Como os logs do FastAPI já são JSON estruturado, é possível filtrar por `level`, `event`, `player_id`, `group_id` etc.
  - Exemplo: `{container="api"} | json | level="error"` — lista todos os erros da API
  - Exemplo: `{container="traefik"} | json | status >= 500` — lista todos os 5xx
- Correlação temporal: ver um pico no gráfico de erros e abrir os logs daquele exato momento com um clique
- Alertas baseados em padrões de log (ex: disparar alerta se `level="error"` aparecer mais de 10 vezes em 5 minutos)

### Quando faz sentido
Quando erros intermitentes difíceis de reproduzir começarem a aparecer, ou quando o volume de usuários crescer e rastrear um problema específico (ex: erro de um player_id específico) se tornar necessário.

### Acesso
Mesmo da Proposta 2 — o Loki é adicionado como nova Data Source no Grafana já existente.

---

## 7. Comparativo final

| | Proposta 1 | Proposta 2 | Proposta 3 |
|---|---|---|---|
| RAM adicional | ~160 MB | ~500 MB | ~750 MB |
| RAM total estimada | ~1 GB (25%) | ~1,35 GB (34%) | ~1,6 GB (40%) |
| Métricas de infra (CPU/RAM/disco) | ❌ | ✅ | ✅ |
| Métricas da API (latência, erros) | ❌ | ✅ | ✅ |
| Alertas de uptime | ✅ | ✅ | ✅ |
| Alertas de recursos | ❌ | ✅ | ✅ |
| Logs em tempo real | ✅ (Dozzle) | ❌ | ✅ (Loki) |
| Logs pesquisáveis e persistidos | ❌ | ❌ | ✅ |
| Correlação métricas + logs | ❌ | ❌ | ✅ |
| Complexidade de setup | Baixa | Média | Alta |
| Risco para a aplicação | Mínimo | Baixo | Baixo |

---

## 8. Arquitetura da Proposta 2 (referência para implementação)

```
VPS (76.13.161.72)
│
├── docker-compose.yml          ← aplicação (api, frontend, traefik)
└── docker-compose.monitoring.yml  ← stack de monitoramento separada
    │
    ├── prometheus               porta interna 9090
    │   ├── scrape: node-exporter :9100
    │   ├── scrape: cadvisor      :8080
    │   └── scrape: api           :8000/metrics
    ├── grafana                  porta interna 3001
    │   └── data source: prometheus
    ├── cadvisor                 porta interna 8080
    ├── node-exporter            porta interna 9100
    └── uptime-kuma              porta interna 3001

Traefik (já existente) roteia:
    grafana.rachao.app   → grafana:3001
    uptime.rachao.app    → uptime-kuma:3001
    prometheus.rachao.app → prometheus:9090  (Basic Auth)
```

A stack de monitoramento ficaria em um `docker-compose.monitoring.yml` separado — isolada da aplicação, sem risco de um `docker compose down` derrubar o monitoramento junto.

---

## 9. Status de implementação

**Proposta 2 implementada.** Validada localmente em 2026-03-18.

### Arquivos entregues

| Arquivo | Descrição |
|---|---|
| `football-api/docker-compose.monitoring.yml` | Stack local (portas expostas) |
| `football-api/docker-compose.monitoring.prod.yml` | Stack produção (via Traefik) |
| `football-api/monitoring/prometheus/prometheus.yml` | Scrape config (api, cadvisor, node-exporter) |
| `football-api/monitoring/grafana/provisioning/datasources/prometheus.yml` | Auto-provisioning do datasource |
| `football-api/.env.monitoring.example` | Template de variáveis para produção |
| `football-api/traefik-dynamic.yml` | Atualizado com rotas grafana/uptime/prometheus + Basic Auth |
| `football-api/app/main.py` | Adicionado `prometheus-fastapi-instrumentator` — expõe `/metrics` |

### Limitação conhecida (ambiente local)

cAdvisor não detecta containers por nome no Docker 29.x com storage driver `overlayfs`. Em produção (VPS com `overlay2`) funciona normalmente. Node Exporter e FastAPI metrics não têm essa limitação.

---

## 10. Checklist de deploy em produção

### Pré-requisitos

- [ ] Todos os arquivos de monitoramento commitados e no repositório
- [ ] DNS configurado (painel Hostinger):
  - [ ] `grafana.rachao.app` → IP do VPS (`76.13.161.72`)
  - [ ] `uptime.rachao.app` → IP do VPS
  - [ ] `prometheus.rachao.app` → IP do VPS _(opcional — acesso restrito por Basic Auth)_

### GitHub Actions

Um novo workflow foi criado: **Deploy Monitoring Stack** (`.github/workflows/deploy-monitoring.yml`).

**Novo secret necessário:**

| Secret | Valor |
|---|---|
| `GRAFANA_ADMIN_PASSWORD` | Senha forte para o admin do Grafana |
| `GRAFANA_ADMIN_USER` | _(opcional)_ — padrão: `admin` |

Os secrets de VPS (`VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`, `VPS_PORT`) já existem. ✅

O workflow existente de deploy da aplicação já copia o `traefik-dynamic.yml` atualizado a cada deploy, incluindo as rotas de monitoramento. ✅

### VPS — setup único via GitHub Actions

**1. Criar o secret no GitHub:**
- Acesse **Settings → Secrets and variables → Actions**
- Adicione `GRAFANA_ADMIN_PASSWORD` com uma senha forte

**2. Executar o workflow:**
- Acesse **Actions → Deploy Monitoring Stack → Run workflow**

O workflow faz automaticamente:
- Copia `docker-compose.monitoring.prod.yml` e `monitoring/` para o VPS
- Cria o `.env.monitoring` com as credenciais
- Sobe a stack com `docker compose up -d`

**3. (Opcional) Trocar a senha do Prometheus Basic Auth** — padrão: `monitoring123`:
```bash
ssh root@76.13.161.72
apt-get install -y apache2-utils
htpasswd -nb admin NOVA_SENHA
# Substitui no traefik-dynamic.yml (sem precisar de redeploy)
sed -i 's|admin:\$apr1\$.*|admin:HASH_GERADO|' /opt/football-manager/traefik-dynamic.yml
```

### Pós-deploy

- [ ] `https://grafana.rachao.app` abre com login _(aguardar ~30s para TLS do Let's Encrypt)_
- [ ] `https://uptime.rachao.app` abre tela de cadastro do primeiro acesso
- [ ] `https://prometheus.rachao.app` pede Basic Auth e mostra UI do Prometheus

**Configurar Uptime Kuma** (primeiro acesso — criar conta admin):
- [ ] Adicionar monitor: `https://rachao.app` — tipo HTTPS, intervalo 60s
- [ ] Adicionar monitor: `https://api.rachao.app/api/v1/health` — tipo HTTPS, intervalo 60s
- [ ] Configurar notificação: Telegram (criar bot via @BotFather, gratuito)

**Importar dashboards no Grafana** (Dashboards → Import):
- [ ] ID `1860` — Node Exporter Full (métricas do host)
- [ ] ID `14282` — Docker cAdvisor (métricas por container)
- [ ] ID `22676` — FastAPI Observability (latência e erros por rota)

**Configurar alertas no Grafana** (Alerting → Alert rules):
- [ ] RAM > 85% por 5 minutos
- [ ] Disco > 80%
- [ ] CPU > 90% por 10 minutos
- [ ] Taxa de erros 5xx > 1% das requisições por 5 minutos
