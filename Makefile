.PHONY: fmt lint tidy gen-docs gen-mocks test-unit tu test-e2e e2e test up-webhook-mock up up-build down restart

fmt:
	go fmt ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

gen-docs:
	swag init -g cmd/geo-alerts/main.go --parseInternal

gen-mocks:
	mockery

test-unit tu:
	go test ./internal/services/... ./internal/utils/...

test-e2e e2e:
	docker compose --env-file .env.test -f docker-compose.test.yaml up -d --build
	go test ./tests -v -count=1 ; \
	EXIT_CODE=$$? ; \
	docker compose -f docker-compose.test.yaml down -v ; \
	exit $$EXIT_CODE

test:
	make test-unit
	make test-e2e

up-webhook-mock:
	go run cmd/webhook-mock/main.go

up:
	docker compose up -d

up-build:
	docker compose up -d --build

down:
	docker compose down

restart:
	docker compose down -v
	docker compose up -d --build