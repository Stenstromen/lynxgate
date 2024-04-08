# LynxGate

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
