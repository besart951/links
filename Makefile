COMPOSE = docker compose --env-file .env -f infra/docker/docker-compose.yml

.PHONY: up down logs build

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

build:
	$(COMPOSE) build
