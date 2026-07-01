.PHONY: debug fe server sync_db dump_db middleware web down clean python help proto-gen dev dev-middleware dev-server dev-down dev-clean test test-all build matrix-env matrix-up matrix-down matrix-clean matrix-logs matrix-ps matrix-wait matrix-test matrix-fe-go matrix-fe-python matrix-fe-java matrix-fe

# 定义脚本路径
SCRIPTS_DIR := ./scripts
BUILD_FE_SCRIPT := $(SCRIPTS_DIR)/build_fe.sh
BUILD_SERVER_SCRIPT := $(SCRIPTS_DIR)/setup/server.sh
SYNC_DB_SCRIPT := $(SCRIPTS_DIR)/setup/db_migrate_apply.sh
DUMP_DB_SCRIPT := $(SCRIPTS_DIR)/setup/db_migrate_dump.sh
SETUP_DOCKER_SCRIPT := $(SCRIPTS_DIR)/setup/docker.sh
SETUP_PYTHON_SCRIPT := $(SCRIPTS_DIR)/setup/python.sh
COMPOSE_FILE := docker/docker-compose-debug.yml
OCEANBASE_COMPOSE_FILE := docker/docker-compose-oceanbase.yml
OCEANBASE_DEBUG_COMPOSE_FILE := docker/docker-compose-oceanbase_debug.yml
MYSQL_SCHEMA := ./docker/volumes/mysql/schema.sql
MYSQL_INIT_SQL := ./docker/volumes/mysql/sql_init.sql
ENV_FILE := ./docker/.env.debug
WEB_ENV_FILE := ./docker/.env
OCEANBASE_ENV_FILE := ./docker/.env.debug
STATIC_DIR := ./bin/resources/static
ES_INDEX_SCHEMA := ./docker/volumes/elasticsearch/es_index_schema
ES_SETUP_SCRIPT := ./docker/volumes/elasticsearch/setup_es.sh

debug: env middleware python server

env:
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "Env file '$(ENV_FILE)' not found, using example env..."; \
		cp ./docker/.env.debug.example $(ENV_FILE); \
	fi

fe:
	@echo "Building frontend..."
	@bash $(BUILD_FE_SCRIPT)

server: env
	@if [ ! -d "$(STATIC_DIR)" ]; then \
		echo "Static directory '$(STATIC_DIR)' not found, building frontend..."; \
		$(MAKE) fe; \
	fi
	@echo "Building and run server..."
	@APP_ENV=debug bash $(BUILD_SERVER_SCRIPT) -start


build_server:
	@echo "Building server..."
	@bash $(BUILD_SERVER_SCRIPT)

sync_db: env
	@echo "Syncing database..."
	@docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) --profile mysql-setup up -d

dump_db: env dump_sql_schema
	@echo "Dumping database..."
	@. $(ENV_FILE); \
	bash $(DUMP_DB_SCRIPT)

sql_init:
	@echo "Init sql data..."
	@docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) --profile mysql-setup up -d

middleware:
	@echo "Start middleware docker environment for opencoze app"
	@docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) --profile middleware up -d --wait

build_docker:
	@echo "Build docker image"
	@docker compose -f $(COMPOSE_FILE) --profile build-server build

web_env:
	@if [ ! -f "$(WEB_ENV_FILE)" ]; then \
		echo "Env file '$(WEB_ENV_FILE)' not found, using example env..."; \
		cp ./docker/.env.example $(WEB_ENV_FILE); \
	fi

web: web_env
	@echo "Start web server in docker"
	@docker compose -f docker/docker-compose.yml --env-file $(WEB_ENV_FILE) up -d

down_web:
	@echo "Stop web server in docker"
	@docker compose -f docker/docker-compose.yml --env-file $(WEB_ENV_FILE) down

down: env
	@echo "Stop all docker containers"
	@docker compose -f $(COMPOSE_FILE) --profile '*' down

clean: down
	@echo "Remove docker containers and volumes data"
	@rm -rf ./docker/data

python:
	@echo "Setting up Python..."
	@bash $(SETUP_PYTHON_SCRIPT)

dump_sql_schema:
	@echo "Dumping mysql schema to $(MYSQL_SCHEMA)..."
	@. $(ENV_FILE); \
	{ echo "SET NAMES utf8mb4;\nCREATE DATABASE IF NOT EXISTS opencoze COLLATE utf8mb4_unicode_ci;"; atlas schema inspect -u $$ATLAS_URL --format "{{ sql . }}" --exclude "atlas_schema_revisions,table_*" | sed 's/CREATE TABLE/CREATE TABLE IF NOT EXISTS/g'; } > $(MYSQL_SCHEMA)
		@sed -i.bak -E 's/(\))[[:space:]]+CHARSET utf8mb4/\1 ENGINE=InnoDB CHARSET utf8mb4/' $(MYSQL_SCHEMA) && rm -f $(MYSQL_SCHEMA).bak
		@sed -i.bak "s/\"/'/g" $(MYSQL_SCHEMA) && rm -f $(MYSQL_SCHEMA).bak
	@cat $(MYSQL_INIT_SQL) >> $(MYSQL_SCHEMA)
	@echo "Dumping mysql schema to helm/charts/opencoze/files/mysql ..."
	@cp $(MYSQL_SCHEMA) ./helm/charts/opencoze/files/mysql/

atlas-hash:
	@echo "Rehash atlas migration files..."
	@(cd ./docker/atlas && atlas migrate hash)

setup_es_index:
	@echo "Setting up Elasticsearch index..."
	@. $(ENV_FILE); \
	bash $(ES_SETUP_SCRIPT) --index-dir $(ES_INDEX_SCHEMA) --docker-host false --es-address "$$ES_ADDR"

oceanbase_env:
	@bash scripts/setup/oceanbase_env.sh debug

oceanbase_debug: oceanbase_env oceanbase_middleware_debug python oceanbase_server_debug

oceanbase_middleware_debug:
	@echo "Starting OceanBase debug middleware..."
	@docker compose -f $(OCEANBASE_DEBUG_COMPOSE_FILE) --env-file $(ENV_FILE) --profile middleware up -d --wait

oceanbase_server_debug:
	@if [ ! -d "$(STATIC_DIR)" ]; then \
		echo "Static directory '$(STATIC_DIR)' not found, building frontend..."; \
		$(MAKE) fe; \
	fi
	@echo "Building and run OceanBase debug server..."
	@APP_ENV=debug bash $(BUILD_SERVER_SCRIPT) -start

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development (lightweight — MySQL + Redis only):"
	@echo "  dev              - Start dev middleware + backend (recommended)."
	@echo "  dev-middleware   - Start MySQL + Redis containers only."
	@echo "  dev-server       - Build and run backend (requires middleware running)."
	@echo "  dev-down         - Stop dev containers."
	@echo "  dev-clean        - Stop containers and delete dev data."
	@echo ""
	@echo "Build & Test:"
	@echo "  build            - Compile backend to bin/superagent."
	@echo "  test             - Run core package tests (pkg/...)."
	@echo "  test-all         - Run all Go tests."
	@echo "  proto-gen        - Generate Go code from api/proto/ definitions."
	@echo ""
	@echo "Full Debug Stack (MySQL + Redis + ES + MinIO + NSQ):"
	@echo "  debug            - Start the full debug environment."
	@echo "  env              - Setup env file (docker/.env.debug)."
	@echo "  fe               - Build frontend assets into bin/resources/static."
	@echo "  server           - Build and run the server binary (debug stack)."
	@echo "  build_server     - Build the server binary without running."
	@echo "  sync_db          - Sync opencoze_latest_schema.hcl to the database."
	@echo "  dump_db          - Dump the database to opencoze_latest_schema.hcl and migrations files."
	@echo "  sql_init         - Init sql data."
	@echo "  dump_sql_schema  - Dump the database schema to sql file."
	@echo "  middleware       - Start middleware containers only (no server)."
	@echo "  web              - Start web docker environment including middlewares."
	@echo "  down             - Stop all debug docker containers."
	@echo "  down_web         - Stop web docker containers."
	@echo "  clean            - Stop containers and remove volume data."
	@echo "  python           - Setup Python environment."
	@echo "  atlas-hash       - Rehash atlas migration files."
	@echo "  setup_es_index   - Setup Elasticsearch index."
	@echo ""
	@echo "OceanBase Commands:"
	@echo "  oceanbase_env    - Setup OceanBase environment file (like 'env')."
	@echo "  oceanbase_debug  - Start OceanBase debug environment (like 'debug')."
	@echo ""
	@echo "  help             - Show this help message."

# Development
dev: dev-middleware dev-server

dev-middleware:
	@echo "Starting dev middleware (MySQL + Redis)..."
	@if ! docker info > /dev/null 2>&1; then \
		echo "Docker daemon not running, please start Docker Desktop manually and retry."; \
		exit 1; \
	fi
	@docker compose -f docker/docker-compose-dev.yml up -d --wait

dev-server:
	@echo "Building and running backend..."
	@cd backend && APP_ENV=dev go run .

dev-down:
	@echo "Stopping dev environment..."
	@docker compose -f docker/docker-compose-dev.yml down

dev-clean: dev-down
	@rm -rf docker/data/mysql-dev
	@echo "Cleaned dev data."

# Testing
test:
	@cd backend && go test ./pkg/... -count=1

test-all:
	@cd backend && go test ./... -count=1

# Build
build:
	@cd backend && go build -o ../bin/superagent .
	@echo "Built: bin/superagent"

# Proto generation
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc
# Install protoc:
#   macOS:  brew install protobuf
#   Linux:  apt-get install -y protobuf-compiler   (or download from github.com/protocolbuffers/protobuf/releases)
#   Both:   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   Both:   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
PROTO_DIR := api/proto
GRPC_GEN_DIR := backend/api/grpc/gen
MODULE := github.com/superagent-ai/superagent-base/backend

proto-gen:
	@echo "Generating Go code from proto files..."
	@for service in agent conversation model tool; do \
		echo "  -> $$service"; \
		protoc \
			--proto_path=$(PROTO_DIR) \
			--go_out=$(GRPC_GEN_DIR) \
			--go_opt=paths=source_relative \
			--go-grpc_out=$(GRPC_GEN_DIR) \
			--go-grpc_opt=paths=source_relative \
			$$service/v1/$$service.proto; \
	done
	@echo "Proto generation complete."

# ── Matrix deployment targets ────────────────────────────────────────

MATRIX_COMPOSE := docker/docker-compose-matrix.yml
MATRIX_ENV     := docker/.env.matrix

matrix-env:
	@if [ ! -f "$(MATRIX_ENV)" ]; then \
		echo "Creating $(MATRIX_ENV) from example..."; \
		cp docker/.env.matrix.example $(MATRIX_ENV); \
		echo "Edit $(MATRIX_ENV) and set your OPENAI_API_KEY before running matrix-up"; \
	fi

matrix-up: matrix-env
	@echo "Starting matrix: Go:8888 Python:8889 Java:8890 ..."
	@docker compose -f $(MATRIX_COMPOSE) --env-file $(MATRIX_ENV) up -d --build

matrix-down:
	@docker compose -f $(MATRIX_COMPOSE) down

matrix-clean:
	@docker compose -f $(MATRIX_COMPOSE) down -v
	@rm -rf docker/data/matrix-mysql

matrix-logs:
	@docker compose -f $(MATRIX_COMPOSE) logs -f

matrix-ps:
	@docker compose -f $(MATRIX_COMPOSE) ps

matrix-wait:
	@echo "Waiting for all three backends to be healthy..."
	@for port in 8888 8889 8890; do \
		echo -n "  Waiting for :$$port "; \
		for i in $$(seq 1 30); do \
			if curl -sf http://localhost:$$port/health > /dev/null 2>&1 || \
			   curl -sf http://localhost:$$port/actuator/health > /dev/null 2>&1; then \
				echo " OK"; break; \
			fi; \
			echo -n "."; sleep 3; \
		done; \
	done

matrix-test: matrix-wait
	@echo "Running matrix tests..."
	@cd tests/matrix && pip install -q -r requirements.txt && pytest -v --tb=short

matrix-fe-go:
	@cd web && VITE_API_BASE=http://localhost:8888 npx vite --port 3501 --mode go

matrix-fe-python:
	@cd web && VITE_API_BASE=http://localhost:8889 npx vite --port 3502 --mode python

matrix-fe-java:
	@cd web && VITE_API_BASE=http://localhost:8890 npx vite --port 3503 --mode java

matrix-fe:
	@echo "Starting 3 frontend instances (Go:3501, Python:3502, Java:3503)..."
	@$(MAKE) matrix-fe-go & \
	 $(MAKE) matrix-fe-python & \
	 $(MAKE) matrix-fe-java & \
	 wait

