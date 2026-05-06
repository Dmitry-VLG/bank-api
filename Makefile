run:
	go run ./cmd/api

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-reset:
	docker compose down -v
	docker compose up -d