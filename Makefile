SHELL := /bin/bash
DOCKER_COMPOSE_FILE := deploy/docker-compose/docker-compose.yml
BINARY_PATH := ./bin/server
DOCKER_IMAGE := 1.1.1.1:5000/demo-task:v1
SLEEP_TIME := 10

.PHONY: init bootstrap mock test build docker swag wire up-deps down-deps

init:
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/golang/mock/mockgen@latest
	go install github.com/swaggo/swag/cmd/swag@latest

bootstrap: up-deps migrate run-server

migrate:
	sleep $(SLEEP_TIME)
	go run ./cmd/migration

run-server:
	nunu run ./cmd/server

mock:
	mockgen -source=internal/service/user.go -destination=test/mocks/service/user.go
	mockgen -source=internal/repository/user.go -destination=test/mocks/repository/user.go
	mockgen -source=internal/repository/repository.go -destination=test/mocks/repository/repository.go

test:
	go test -coverpkg=./internal/handler,./internal/service,./internal/repository -coverprofile=./coverage.out ./test/server/...
	go tool cover -html=./coverage.out -o coverage.html

build:
	go build -ldflags="-s -w" -o $(BINARY_PATH) ./cmd/server

docker:
	docker compose -f $(DOCKER_COMPOSE_FILE) up -d
	docker build -f deploy/build/Dockerfile -t $(DOCKER_IMAGE) .
	docker run --rm --network local -p 8000:8000 -i $(DOCKER_IMAGE)

swag:
	swag init -g cmd/server/main.go -o ./docs --parseDependency

wire:
	wire ./cmd/server

up-deps:
	docker compose -f $(DOCKER_COMPOSE_FILE) up -d

down-deps:
	docker compose -f $(DOCKER_COMPOSE_FILE) down
