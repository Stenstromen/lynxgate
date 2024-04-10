# LynxGate

![logo](lynxgate_logo.webp)

## Dev

### MySQL Encryption Key

```bash
openssl rand -hex 16 > .secret
```

### DB

```bash
[ -d "$PWD/dbdata" ] || mkdir -p "$PWD/dbdata" && \
docker run --rm -d \
-v "$PWD/dbdata:/var/lib/mysql" \
-e MYSQL_ROOT_PASSWORD="passwd" \
-e MYSQL_DATABASE="lynxgate_test" \
-p 3306:3306 \
mariadb:latest
```

### App

```bash
MYSQL_ENCRYPTION_KEY=$(cat .secret) \
MYSQL_DSN="root:passwd@tcp(127.0.0.1:3306)/lynxgate_test" \
go run .
```

## API Docs

### Authentication

No built in authentication

### Base URL

All requests start at root `/`

### Endpoints

#### Create a API Key

- **HTTP Method:** POST
- **URL:** `/tokens`
- **Description:** Creates an API token
- **Headers:**
  - `Content-Type: application/json`
- **Request Body Example:**

  ```json
  {
   "accountID": "Protected_App1",
   "quota": 100
  }
  ```
