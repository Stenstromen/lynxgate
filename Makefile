.PHONY: build compose-up compose-down test-deps test clean

NETWORK_NAME = testnetwork
DB_CONTAINER = test-mariadb
APP_CONTAINER = test-lynxgate
DB_PASSWORD = testpass123

build:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags='-w -s -extldflags "-static"' -o ./lynxgate

compose-up:
	podman-compose build --no-cache
	podman-compose up

compose-down:
	podman-compose down

test-deps:
	@which podman >/dev/null 2>&1 || (echo "❌ podman is required but not installed. Aborting." && exit 1)
	@which curl >/dev/null 2>&1 || (echo "❌ curl is required but not installed. Aborting." && exit 1)
	@which jq >/dev/null 2>&1 || (echo "❌ jq is required but not installed. Aborting." && exit 1)

test: test-deps
	@echo "ℹ️ Creating podman network..."
	podman network create $(NETWORK_NAME) || true

	@echo "ℹ️ Starting MariaDB container..."
	podman run -d --name $(DB_CONTAINER) \
		--network $(NETWORK_NAME) \
		-e MYSQL_ROOT_PASSWORD=$(DB_PASSWORD) \
		-e MYSQL_DATABASE=lynxgate \
		-e MYSQL_USER=lynxgate \
		-e MYSQL_PASSWORD=$(DB_PASSWORD) \
		docker.io/library/mariadb:latest

	@echo "ℹ️ Building application container..."
	podman build -t $(APP_CONTAINER) .

	@echo "ℹ️ Waiting for MariaDB to be ready..."
	sleep 5

	@echo "ℹ️ Starting application container..."
	podman run -d --name $(APP_CONTAINER) \
		--network $(NETWORK_NAME) \
		-p 8080:8080 \
		-e MYSQL_DSN="lynxgate:$(DB_PASSWORD)@tcp($(DB_CONTAINER):3306)/lynxgate" \
		-e MYSQL_ENCRYPTION_KEY="7AE49A19B3C844BDB68E460D9224A5D0" \
		-e CURRENT_DATE="2024-03-01" \
		$(APP_CONTAINER)

	@echo "ℹ️ Running integration tests..."
	./integration_test.sh

clean:
	@echo "ℹ️ Cleaning up containers and volumes..."
	podman stop $(APP_CONTAINER) $(DB_CONTAINER) || true
	podman rm -v $(APP_CONTAINER) $(DB_CONTAINER) || true
	podman network rm $(NETWORK_NAME) || true