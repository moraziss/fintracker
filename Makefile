include .env
export

.PHONY: run migrate-up migrate-down migrate-down-all migrate-version migrate-create

run:
	go run ./cmd/api

migrate-up:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

migrate-down-all:
	migrate -path db/migrations -database "$(DATABASE_URL)" down -all

migrate-version:
	migrate -path db/migrations -database "$(DATABASE_URL)" version

migrate-create:
	migrate create -ext sql -dir db/migrations -seq $(name)