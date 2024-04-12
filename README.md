# LynxGate

![logo](lynxgate_logo.webp)

## Overview

LynxGate is a simple API Gateway that provides API key and quota management.

For example, the `nginx.ingress.kubernetes.io/auth-url` annotation in Kubernetes Nginx Ingress Controller can be used to validate the API token.

Instead of using a third-party service like Kong, Tyk, or Apigee, **LynxGate** can be used to manage API keys and quotas.

## Features

- API Key Management
- Quota Management
- API Token Validation
- API Token Creation
- API Token Deletion
- API Token Listing
- API Token Retrieval
- API Token Quota Usage

## Requirements

- Container Runtime (Kubernetes, Podman, Docker etc)
- MySQL Endpoint

## Deployment

### Generate MySQL Encryption Key

The `token` row in the `tokens` table is encrypted using the default AES algorithm for the database endpoint. The encryption key is stored in the `.secret` file.

```bash
openssl rand -hex 16 > .secret
```

### Podman

```bash
podman run --rm -d \
-e MYSQL_ENCRYPTION_KEY=$(cat .secret) \
-e MYSQL_DSN="USERNAME:PASSWORD@tcp(HOST:PORT)/DBNAME" \
-p 80:8080 \
ghcr.io/stenstromen/lynxgate:latest
```

### Example requests

#### Create API Token

```bash
curl -X POST http://localhost/tokens \
-H "Content-Type: application/json" \
-d '{"accountID": "Protected_App1", "quota": 100}'
```

##### `/tokens` Response

```json
{
  "accountID": "Protected_App1",
  "token": "051e74a7469943e7a6fde4ea85a458aa",
  "quota": 100
}
```

#### Test API Token

```bash
curl -X GET http://localhost/validate \
-H "Authorization: 051e74a7469943e7a6fde4ea85a458aa"
```

##### `/validate` Response

```console
200 OK
```

## API Docs

### Authentication

No built in authentication

### Base URL

All requests start at root `/`

### Endpoints

#### Validation Endpoint

- **HTTP Method:** GET
- **URL:** `/validate`
- **Description:** Validates the API token (for use in the API Gateway)
- **Headers:**

  - `Content-Type: application/json`
  - `Authorization: {token}`

- **Success Response:**

  - **Code:** 200

#### Create API Key

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

- **Success Response:**

  - **Code:** 201
  - **Content:**

    ```json
    {
      "account_id": "Protected_App1",
      "token": "051e74a7469943e7a6fde4ea85a458aa",
      "quota": 100
    }
    ```

#### Get API Key

- **HTTP Method:** GET
- **URL:** `/tokens/{accountID}`
- **Description:** Get an API token
- **Headers:**

  - `Content-Type: application/json`

- **Success Response:**

  - **Code:** 200
  - **Content:**

    ```json
    {
      "account_id": "Protected_App1",
      "token": "051e74a7469943e7a6fde4ea85a458aa",
      "quota": 100,
      "quota_usage": 22
    }
    ```

#### Delete API Key

- **HTTP Method:** DELETE
- **URL:** `/tokens/{accountID}`
- **Description:** Delete an API token
- **Headers:**

  - `Content-Type: application/json`

- **Success Response:**

  - **Code:** 200

#### List API Keys

- **HTTP Method:** GET
- **URL:** `/tokens`
- **Description:** List all API tokens
- **Headers:**

  - `Content-Type: application/json`

- **Success Response:**

  - **Code:** 200
  - **Content:**

  ```json
  [
    {
      "account_id": "Protected_App1",
      "token": "051e74a7469943e7a6fde4ea85a458aa",
      "quota": 100,
      "quota_usage": 22
    },
    {
      "account_id": "Protected_App2",
      "token": "051e74a7469943ebabfde4ea85a458aa",
      "quota": 23,
      "quota_usage": 0
    },
    {
      "account_id": "Protected_App3",
      "token": "051e74a746931337a6fde4ea85a458aa",
      "quota": 33,
      "quota_usage": 10
    }
  ]
  ```

## Dev

### MySQL Encryption Key

```bash
openssl rand -hex 16 > .secret
```

### DB

```bash
[ -d "$PWD/dbdata" ] || mkdir -p "$PWD/dbdata" && \
podman run --rm -d \
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
