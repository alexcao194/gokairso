.PHONY: sqlc docker-up docker-down docker-build docker-logs migrate-up migrate-down migrate-status migrate-create tidy build swagger

# Load environment variables from .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# ---- Code Generation (SQLC) ----
sqlc:
	sqlc generate

# ---- Docker ----
dev:
	docker compose -p kairso-$(shell basename $(CURDIR)) down && docker compose -p kairso-$(shell basename $(CURDIR)) up --build -d
dev-down:
	docker compose -p kairso-$(shell basename $(CURDIR)) down
log:
	docker compose -p kairso-$(shell basename $(CURDIR)) logs -f --tail 100 app

# ---- Database Migration (goose) ----
GOOSE_DRIVER ?= postgres
GOOSE_DBSTRING ?= user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) host=localhost port=$(DB_PORT) sslmode=disable
GOOSE_MIGRATION_DIR ?= db/migrations

migrate-up:
	GOOSE_DRIVER="$(GOOSE_DRIVER)" GOOSE_DBSTRING="$(GOOSE_DBSTRING)" GOOSE_MIGRATION_DIR="$(GOOSE_MIGRATION_DIR)" goose up

migrate-down:
	GOOSE_DRIVER="$(GOOSE_DRIVER)" GOOSE_DBSTRING="$(GOOSE_DBSTRING)" GOOSE_MIGRATION_DIR="$(GOOSE_MIGRATION_DIR)" goose down

migrate-status:
	GOOSE_DRIVER="$(GOOSE_DRIVER)" GOOSE_DBSTRING="$(GOOSE_DBSTRING)" GOOSE_MIGRATION_DIR="$(GOOSE_MIGRATION_DIR)" goose status

migrate-create:
	GOOSE_DRIVER="$(GOOSE_DRIVER)" GOOSE_DBSTRING="$(GOOSE_DBSTRING)" GOOSE_MIGRATION_DIR="$(GOOSE_MIGRATION_DIR)" goose create $(name) sql

swagger:
	swag init -g cmd/server/main.go --parseInternal --parseDependency --st

# ---- Utility ----
tidy:
	go mod tidy

build:
	go build -buildvcs=false -o tmp/server ./cmd/server
