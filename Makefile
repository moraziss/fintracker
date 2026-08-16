include .env
export

migrate-up:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	migrate create -ext sql -dir db/migrations -seq $(name)