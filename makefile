PROJECT_NAME := gophermart
ENV_FILE := .env

build:
	go build -o ./cmd/${PROJECT_NAME}/ ./cmd/${PROJECT_NAME}

test:
	go test ./...

compose-up:
	docker compose --env-file $(ENV_FILE) up --build -d

compose-down:
	docker compose --env-file $(ENV_FILE) down

compose-restart:
	compose-down
	compose-up