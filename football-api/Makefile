.PHONY: up up-bg down down-clean logs shell test db-connect adminer

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
	docker compose exec api pytest tests/ -v

# ── Health ────────────────────────────────────────────────────

health:
	@curl -s http://localhost:8000/health | python3 -m json.tool

docs:
	@open http://localhost:8000/docs 2>/dev/null || xdg-open http://localhost:8000/docs
