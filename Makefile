SHELL := /bin/bash
DB_URL ?= exam:examtest@tcp(127.0.0.1:53307)/exam_eval_test?parseTime=true&multiStatements=true
IMAGE ?= secure-exam-eval-backend
TAG ?= dev

.PHONY: help dev up down logs test test-unit test-api build seed fmt vet compose-test tf-plan tf-apply clean

help:
	@grep -E '^[a-z-]+:.*?##' $(MAKEFILE_LIST) | sed 's/:.*##/ — /'

up: ## Start the whole stack in Docker
	docker compose up -d --build

down: ## Stop the stack
	docker compose down

logs: ## Tail backend logs
	docker compose logs -f backend

seed: ## Load demo cohort into the running stack
	docker compose --profile seed run --rm --build seed

dev: ## Run backend + vite locally (needs local Postgres)
	cd backend && DATABASE_URL=$(DB_URL) go run ./cmd/server & \
	cd frontend && npm run dev

test-unit: ## Fast unit tests, no database
	cd backend && go test ./internal/grading/...

test-api: ## Full API suite against a throwaway Postgres
	@docker rm -f examtest-db >/dev/null 2>&1 || true
	docker run -d --name examtest-db -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=exam_eval_test \
		-e MYSQL_USER=exam -e MYSQL_PASSWORD=examtest -p 53307:3306 mysql:8.4 >/dev/null
	@for i in $$(seq 1 40); do docker exec examtest-db mysqladmin ping -h127.0.0.1 -uroot -proot >/dev/null 2>&1 && break; sleep 2; done
	cd backend && TEST_DATABASE_URL='$(DB_URL)' go test ./... ; status=$$?; \
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
	@sleep 8; curl -fsS http://localhost:8090/healthz && echo " <- stack healthy"
	docker compose down

tf-plan: ## Plan AWS infrastructure
	cd infra/terraform && terraform init -input=false && terraform plan

tf-apply: ## Apply AWS infrastructure
	cd infra/terraform && terraform init -input=false && terraform apply

clean:
	docker compose down -v
