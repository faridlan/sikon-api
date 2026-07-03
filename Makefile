# ==============================================================================
# Environment Variables Dari File .env
# ==============================================================================
include .env
export
# ==============================================================================
# Database Commands
# ==============================================================================

# 1. Menjalankan container PostgreSQL di background
postgres:
	docker run --name omnilibrary-db -e POSTGRES_USER=$(DB_USER) -e POSTGRES_PASSWORD=$(DB_PASSWORD) -p $(DB_PORT):5432 -d postgres:15-alpine

# 2. Membuat database baru di dalam container
createdb:
	docker exec -it omnilibrary-db createdb --username=$(DB_USER) --owner=$(DB_USER) $(DB_NAME)

# 3. Menghapus database (Hati-hati!)
dropdb:
	docker exec -it omnilibrary-db dropdb $(DB_NAME)

# ==============================================================================
# Migration Commands
# ==============================================================================

# 4. Menjalankan migrasi UP (membuat tabel)
migrateup:
	migrate -path db/migrations -database $(DB_URL) -verbose up

# 5. Menjalankan migrasi DOWN (menghapus tabel)
migratedown:
	migrate -path db/migrations -database $(DB_URL) -verbose down

migrateforce:
	migrate -path db/migrations -database $(DB_URL) force $(V)


# ==============================================================================
# Swaggo Commands
# ==============================================================================

swagup:
	swag init -g cmd/api/main.go --parseDependency --parseInternal


# ==============================================================================
# Mockery Commands
# ==============================================================================

mockup:
	mockery --dir=internal/domain --all --output=internal/domain/mocks --outpkg=mocks 

# ==============================================================================
# .PHONY memastikan make tidak bentrok dengan nama folder/file yang kebetulan sama
# ==============================================================================
.PHONY: postgres createdb dropdb migrateup migratedown

.PHONY: migrate-create

# Perintah untuk membuat migrasi baru
# Cara pakai: make migrate-create name=nama_migrasi_kamu
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: Kamu harus memasukkan nama migrasi. Contoh: make migrate-create name=create_users_table"; \
		exit 1; \
	fi
	migrate create -ext sql -dir db/migrations -seq $(name)