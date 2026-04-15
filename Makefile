GO_PACKAGES := ./...
GO_FILES := $(shell rg --files -g '*.go')

.PHONY: fmt test build run-api run-worker run-cli migrate compose-up compose-down web-install web-dev web-build smoke

fmt:
	gofmt -w $(GO_FILES)

test:
	go test $(GO_PACKAGES)

build:
	go build ./cmd/api
	go build ./cmd/worker
	go build ./cmd/cli
	go build ./cmd/migrate

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

run-cli:
	go run ./cmd/cli

migrate:
	go run ./cmd/migrate

compose-up:
	docker compose -f deploy/docker/docker-compose.yml up -d

compose-down:
	docker compose -f deploy/docker/docker-compose.yml down -v

web-install:
	cd web && pnpm install

web-dev:
	cd web && pnpm dev

web-build:
	cd web && pnpm build

smoke:
	./scripts/smoke.sh
