.PHONY: run build test docker-up docker-down

run:
	go run cmd/gateway/main.go

build:
	go build -o bin/gateway cmd/gateway/main.go

test:
	go test ./...

docker-up:
	docker compose -f deployments/docker/docker-compose.yml up -d

docker-down:
	docker compose -f deployments/docker/docker-compose.yml down