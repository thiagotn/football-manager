.PHONY: up rebuild down down-clean logs

up:
	docker compose up --build

rebuild:
	docker compose build --no-cache && docker compose up

down:
	docker compose down

down-clean:
	docker compose down -v

logs:
	docker compose logs -f api frontend
