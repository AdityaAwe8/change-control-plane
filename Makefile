GO_PACKAGES := ./...
GO_FILES := $(shell if command -v rg >/dev/null 2>&1; then rg --files -g '*.go' cmd internal pkg test; else find cmd internal pkg test -name '*.go' -type f 2>/dev/null; fi)
PYTHON ?= python3
CCP_DB_DSN ?= postgres://postgres:postgres@localhost:15432/change_control_plane?sslmode=disable
CCP_REDIS_ADDR ?= localhost:16379
CCP_NATS_URL ?= nats://localhost:14222
CCP_API_PORT ?= 8080

.PHONY: fmt test test-go test-python verify proof-contract proof-harness proof-live-preflight proof-live-verify proof-live-validate release-readiness build run-api run-worker run-cli migrate compose-up compose-up-full compose-down web-install web-dev web-build web-typecheck web-e2e smoke reference-pilot-up reference-pilot-down reference-pilot-verify reference-pilot-validate

fmt:
	gofmt -w $(GO_FILES)

test: test-go test-python

test-go:
	go test $(GO_PACKAGES)

test-python:
	$(PYTHON) -m unittest discover -s python/tests -v

verify: test web-typecheck web-build

proof-contract:
	go test ./internal/app -run 'TestOpenAPI'

proof-harness:
	go test ./internal/app ./internal/delivery ./internal/verification ./internal/integrations -run 'Test(GitHubAppWebhookRegistrationSyncRepairsExistingHostedWebhook|GitLabWebhookRegistrationSyncRepairsExistingHostedWebhook|KubernetesAndPrometheusIntegrationRoutesHonorConfiguredAuthHeadersAndPaths|CreateGitHubAppInstallationToken|GitLabClientConnectionDiscoveryAndMergeRequestChanges|ParseGitLabMergeRequestWebhookNormalizesChange|KubernetesProvider.*|PrometheusProvider.*)'

proof-live-preflight:
	./scripts/live-proof-preflight.sh

proof-live-verify:
	./scripts/live-proof-verify.sh

proof-live-validate:
	./scripts/live-proof-validate.sh

release-readiness:
	./scripts/release-readiness.sh

web-e2e:
	cd web && pnpm test:e2e

build:
	go build ./cmd/api
	go build ./cmd/worker
	go build ./cmd/cli
	go build ./cmd/migrate

run-api:
	CCP_DB_DSN='$(CCP_DB_DSN)' CCP_REDIS_ADDR='$(CCP_REDIS_ADDR)' CCP_NATS_URL='$(CCP_NATS_URL)' CCP_API_PORT='$(CCP_API_PORT)' ./scripts/run-with-local-env.sh go run ./cmd/api

run-worker:
	CCP_DB_DSN='$(CCP_DB_DSN)' CCP_REDIS_ADDR='$(CCP_REDIS_ADDR)' CCP_NATS_URL='$(CCP_NATS_URL)' ./scripts/run-with-local-env.sh go run ./cmd/worker

run-cli:
	./scripts/run-with-local-env.sh go run ./cmd/cli

migrate:
	CCP_DB_DSN='$(CCP_DB_DSN)' ./scripts/run-with-local-env.sh go run ./cmd/migrate

compose-up:
	docker compose -f deploy/docker/docker-compose.yml up -d --wait postgres redis nats

compose-up-full:
	docker compose -f deploy/docker/docker-compose.yml up -d --build

compose-down:
	docker compose -f deploy/docker/docker-compose.yml down -v

web-install:
	cd web && pnpm install

web-dev:
	cd web && pnpm dev

web-typecheck:
	cd web && pnpm typecheck

web-build:
	cd web && pnpm build

smoke:
	./scripts/smoke.sh

reference-pilot-up:
	./scripts/reference-pilot-up.sh

reference-pilot-down:
	./scripts/reference-pilot-down.sh

reference-pilot-verify:
	./scripts/reference-pilot-verify.sh

reference-pilot-validate:
	./scripts/reference-pilot-validate.sh
