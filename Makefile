.PHONY: init
init:
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/golang/mock/mockgen@latest
	go install github.com/swaggo/swag/cmd/swag@latest

.PHONY: bootstrap
bootstrap:
	docker compose -f deploy/docker-compose/docker-compose.yml up -d
	sleep 10
	go run ./cmd/migration
	nunu run ./cmd/server

.PHONY: mock
mock:
	mockgen -source=internal/service/user.go -destination test/mocks/service/user.go
	mockgen -source=internal/repository/user.go -destination test/mocks/repository/user.go
	mockgen -source=internal/repository/repository.go -destination test/mocks/repository/repository.go

.PHONY: test
test:
	go test -coverpkg=./internal/handler,./internal/service,./internal/repository -coverprofile=./coverage.out ./test/server/...
	go tool cover -html=./coverage.out -o coverage.html

.PHONY: build
build:
	go build -ldflags="-s -w" -o ./bin/server ./cmd/server

.PHONY: docker
docker:
	docker compose -f deploy/docker-compose/docker-compose.yml up -d
	docker build -f deploy/build/Dockerfile -t 1.1.1.1:5000/demo-task:v1 .
	docker run --rm --network local -p 8000:8000 -i 1.1.1.1:5000/demo-task:v1

.PHONY: swag
swag:
	swag init  -g cmd/server/main.go -o ./docs --parseDependency

.PHONY: wire
wire:
	wire ./cmd/server

.PHONY: up-deps
up-deps:
	docker compose -f deploy/docker-compose/docker-compose.yml up -d

.PHONY: down-deps
down-deps:
	docker compose -f deploy/docker-compose/docker-compose.yml down
