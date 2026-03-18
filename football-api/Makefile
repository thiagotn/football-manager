.PHONY: up up-bg down down-clean logs shell test test-unit db-connect adminer monitoring monitoring-down monitoring-logs

# ── Ambiente ─────────────────────────────────────────────────

up:
	docker compose up --build

up-bg:
	docker compose up --build -d && docker compose logs -f api

down:
	docker compose down

down-clean:
	docker compose down -v

logs:
	docker compose logs -f api frontend

shell:
	docker compose exec api bash

# ── Banco ─────────────────────────────────────────────────────

db-connect:
	docker compose exec postgres psql -U postgres -d football

db-reset:
	docker compose down -v && docker compose up postgres -d --wait
	@echo "Banco recriado ✓"

adminer:
	docker compose --profile tools up adminer -d
	@echo "Adminer: http://localhost:8080"
	@echo "Server: postgres | User: postgres | Password: football123"

# ── Testes ────────────────────────────────────────────────────

test:
	docker compose run --rm api poetry run pytest tests/unit/ -v

test-unit: test

test-cov:
	docker compose run --rm api poetry run pytest tests/unit/ --cov=app --cov-report=term-missing -q

# ── Health ────────────────────────────────────────────────────

health:
	@curl -s http://localhost:8000/health | python3 -m json.tool

docs:
	@open http://localhost:8000/docs 2>/dev/null || xdg-open http://localhost:8000/docs

# ── Monitoramento ──────────────────────────────────────────────

monitoring:
	docker compose -f docker-compose.monitoring.yml up -d
	@echo ""
	@echo "Stack de monitoramento iniciada:"
	@echo "  Grafana:      http://localhost:3001  (admin / admin)"
	@echo "  Prometheus:   http://localhost:9090"
	@echo "  Uptime Kuma:  http://localhost:3002"
	@echo "  cAdvisor:     http://localhost:8081"
	@echo ""
	@echo "Dashboards recomendados (importar no Grafana por ID):"
	@echo "  Node Exporter Full:  1860"
	@echo "  Docker cAdvisor:     14282"
	@echo "  FastAPI:             16110"

monitoring-down:
	docker compose -f docker-compose.monitoring.yml down

monitoring-logs:
	docker compose -f docker-compose.monitoring.yml logs -f prometheus grafana
