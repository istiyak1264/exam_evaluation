SHELL := /bin/bash

# Credentials live in .env only; the dev/test targets read them from here.
-include .env
export MYSQL_PASSWORD MYSQL_ROOT_PASSWORD JWT_SECRET BOOTSTRAP_ADMIN_EMAIL \
	BOOTSTRAP_ADMIN_PASSWORD SEED_PASSWORD ALLOWED_ORIGINS APP_ENV \
	WEB_PORT DB_PORT STORAGE_DRIVER

DB_PORT ?= 53306
# Local `make dev`: talk to the compose db, not a throwaway test database.
DB_URL ?= exam:$(MYSQL_PASSWORD)@tcp(127.0.0.1:$(DB_PORT))/exam_eval?parseTime=true&multiStatements=true
# Throwaway database used by `make test-api`.
TEST_DB_PORT ?= 53307
TEST_DB_URL ?= exam:examtest@tcp(127.0.0.1:$(TEST_DB_PORT))/exam_eval_test?parseTime=true&multiStatements=true

IMAGE ?= secure-exam-eval-backend
TAG ?= dev

.PHONY: help dev up down logs test test-unit test-api build seed fmt vet compose-test tf-plan tf-apply clean db-reset

help:
	@grep -hE '^[a-z-]+:.*?##' $(MAKEFILE_LIST) | sed 's/:.*##/ — /'

up: ## Start the whole stack in Docker
	docker compose up -d --build

down: ## Stop the stack
	docker compose down

logs: ## Tail backend logs
	docker compose logs -f backend

seed: ## Load demo cohort into the running stack
	docker compose --profile seed run --rm seed

dev: ## Run backend + vite locally against the compose db
	@test -n "$(MYSQL_PASSWORD)" || { echo "MYSQL_PASSWORD is empty — copy .env.example to .env first"; exit 1; }
	cd backend && DATABASE_URL='$(DB_URL)' go run ./cmd/server & \
	cd frontend && npm run dev

test-unit: ## Fast unit tests, no database
	cd backend && go test ./internal/grading/...

test-api: ## Full API suite against a throwaway MySQL container
	@docker rm -f examtest-db >/dev/null 2>&1 || true
	docker run -d --name examtest-db -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=exam_eval_test \
		-e MYSQL_USER=exam -e MYSQL_PASSWORD=examtest -p $(TEST_DB_PORT):3306 mysql:8.4 >/dev/null
	@for i in $$(seq 1 40); do docker exec examtest-db mysqladmin ping -h127.0.0.1 -uroot -proot >/dev/null 2>&1 && break; sleep 2; done
	cd backend && TEST_DATABASE_URL='$(TEST_DB_URL)' go test ./... ; status=$$?; \
	docker rm -f examtest-db >/dev/null; exit $$status

test: test-unit test-api ## Everything

fmt:
	cd backend && gofmt -w ./cmd ./internal

vet:
	cd backend && go vet ./...

build: ## Build production images
	docker build -t $(IMAGE):$(TAG) ./backend
	docker build -t secure-exam-eval-web:$(TAG) ./frontend

compose-test: ## Boot the stack, hit health, tear down
	docker compose up -d --build
	@for i in $$(seq 1 30); do curl -fsS http://localhost:$${WEB_PORT:-8090}/healthz >/dev/null 2>&1 && break; sleep 2; done
	@curl -fsS http://localhost:$${WEB_PORT:-8090}/healthz && echo " <- stack healthy"
	docker compose down

tf-plan: ## Plan AWS infrastructure
	cd infra/terraform && terraform init -input=false && terraform plan

tf-apply: ## Apply AWS infrastructure
	cd infra/terraform && terraform init -input=false && terraform apply

db-reset: ## Destroy the database volume and recreate it from .env (data is lost)
	docker compose down -v
	docker compose up -d --build

clean:
	docker compose down -v
