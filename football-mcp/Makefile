.PHONY: install dev register unregister list check test test-cov

RACHAO_API_URL ?= https://api.rachao.app/api/v1

# ── Instalação ────────────────────────────────────────────────

install:
	uv pip install -e ".[dev]"

# ── Execução local (stdio) ────────────────────────────────────

dev:
	@if [ -z "$(RACHAO_TOKEN)" ]; then \
		echo "Erro: defina RACHAO_TOKEN=<jwt>"; exit 1; \
	fi
	RACHAO_TOKEN=$(RACHAO_TOKEN) \
	RACHAO_API_URL=$(RACHAO_API_URL) \
	python -m rachao_mcp

# ── Integração com Claude CLI ─────────────────────────────────

register:
	@if [ -z "$(RACHAO_TOKEN)" ]; then \
		echo "Uso: make register RACHAO_TOKEN=<jwt>"; exit 1; \
	fi
	claude mcp add rachao \
		--command "python" \
		--args "-m rachao_mcp" \
		--env RACHAO_TOKEN="$(RACHAO_TOKEN)" \
		--env RACHAO_API_URL="$(RACHAO_API_URL)"
	@echo ""
	@echo "MCP registrado. Verifique com: make list"

unregister:
	claude mcp remove rachao
	@echo "MCP removido."

list:
	claude mcp list

# ── Verificação rápida sem Claude CLI ────────────────────────

check:
	@if [ -z "$(RACHAO_TOKEN)" ]; then \
		echo "Uso: make check RACHAO_TOKEN=<jwt>"; exit 1; \
	fi
	RACHAO_TOKEN=$(RACHAO_TOKEN) \
	RACHAO_API_URL=$(RACHAO_API_URL) \
	python -c "import asyncio; from rachao_mcp.client import api; print(asyncio.run(api.get('/groups')))"

# ── Testes ────────────────────────────────────────────────────

test:
	pytest tests/ -v

test-cov:
	pytest tests/ --cov=rachao_mcp --cov-report=term-missing -q
