#!/bin/bash
set -e

APP_CONTAINER="test-lynxgate"
TEST_MESSAGE="Hello, this is a test message!"
APP_IP="localhost"
DB_CONTAINER="test-mariadb"
DB_PASSWORD="testpass123"
NETWORK_NAME="testnetwork"

# Wait for the application to be ready with connection check
echo "ℹ️ Waiting for application to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0
while ! curl -s "http://$APP_IP:8080/ready" > /dev/null 2>&1; do
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        fail "Timeout waiting for application to start"
    fi
    echo "ℹ️ Waiting... ($(($RETRY_COUNT + 1))/$MAX_RETRIES)"
    sleep 2
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

echo "✅ Application is ready"

# Function to check if test failed
fail() {
    echo "❌ Test failed: $1"
    exit 1
}

# Test non-existent key
echo "ℹ️ Testing non-existent authorization..."
RESPONSE=$(curl -s -w "%{http_code}" \
    -H "Authorization: invalid_key_123" \
    "http://$APP_IP:8080/validate")

if [[ $RESPONSE != *"401"* ]]; then
    fail "Non-existent key should return 401, got: $RESPONSE"
fi

echo "✅ Non-existent key properly rejected"

# Test quota limits
echo "ℹ️ Creating token with quota of 3..."
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"accountID": "QuotaTest", "quota": 3}' \
    "http://$APP_IP:8080/tokens")

QUOTA_TOKEN=$(echo $RESPONSE | jq -r .token)

if [ -z "$QUOTA_TOKEN" ]; then
    fail "Failed to create quota test token"
fi

echo "✅ Created quota test token: $QUOTA_TOKEN"

# Test quota usage
for i in {1..3}; do
    echo "ℹ️ Quota test attempt $i/3..."
    RESPONSE=$(curl -s -w "%{http_code}" \
        -H "Authorization: $QUOTA_TOKEN" \
        "http://$APP_IP:8080/validate")
    
    if [[ $RESPONSE != *"200"* ]]; then
        fail "Quota validation $i should succeed, got: $RESPONSE"
    fi
done

echo "ℹ️ Testing quota exceeded..."
RESPONSE=$(curl -s -w "%{http_code}" \
    -H "Authorization: $QUOTA_TOKEN" \
    "http://$APP_IP:8080/validate")

if [[ $RESPONSE != *"429"* ]]; then
    fail "Quota exceeded should return 429, got: $RESPONSE"
fi

echo "✅ Quota limits working correctly"

# Create second token and verify both exist
echo "ℹ️ Creating second token..."
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"accountID": "SecondTest", "quota": 100}' \
    "http://$APP_IP:8080/tokens")

SECOND_TOKEN=$(echo $RESPONSE | jq -r .token)

echo "ℹ️ Verifying both tokens exist..."
TOKENS_LIST=$(curl -s "http://$APP_IP:8080/tokens")
QUOTA_TOKEN_EXISTS=$(echo $TOKENS_LIST | jq -r '.[] | select(.account_id=="QuotaTest")')
SECOND_TOKEN_EXISTS=$(echo $TOKENS_LIST | jq -r '.[] | select(.account_id=="SecondTest")')

if [ -z "$QUOTA_TOKEN_EXISTS" ] || [ -z "$SECOND_TOKEN_EXISTS" ]; then
    fail "Not all tokens found in listing"
fi

echo "✅ Both tokens exist in listing"

# Delete both tokens
echo "ℹ️ Deleting QuotaTest token..."
curl -s -X DELETE "http://$APP_IP:8080/tokens/QuotaTest"

echo "ℹ️ Deleting SecondTest token..."
curl -s -X DELETE "http://$APP_IP:8080/tokens/SecondTest"

# Verify tokens are deleted
TOKENS_LIST=$(curl -s "http://$APP_IP:8080/tokens")
REMAINING_TOKENS=$(echo $TOKENS_LIST | jq length)

if [ "$REMAINING_TOKENS" != "0" ]; then
    fail "Tokens were not properly deleted. $REMAINING_TOKENS tokens remain"
fi

echo "✅ Tokens successfully deleted"

echo "ℹ️ Testing quota reset..."
echo "ℹ️ Creating token with quota of 1..."
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"accountID": "QuotaResetTest", "quota": 1}' \
    "http://$APP_IP:8080/tokens")

RESET_TOKEN=$(echo $RESPONSE | jq -r .token)

echo "ℹ️ Using up the quota..."
RESPONSE=$(curl -s -w "%{http_code}" \
    -H "Authorization: $RESET_TOKEN" \
    "http://$APP_IP:8080/validate")

if [[ $RESPONSE != *"200"* ]]; then
    fail "First validation should succeed, got: $RESPONSE"
fi

# Verify quota is exhausted
RESPONSE=$(curl -s -w "%{http_code}" \
    -H "Authorization: $RESET_TOKEN" \
    "http://$APP_IP:8080/validate")

if [[ $RESPONSE != *"429"* ]]; then
    fail "Quota should be exhausted, expected 429, got: $RESPONSE"
fi

# Simulate month rollover by restarting the app with a different date
echo "ℹ️ Simulating month rollover..."
podman stop "${APP_CONTAINER}"
podman rm "${APP_CONTAINER}"
podman run -d --name "${APP_CONTAINER}" \
    --restart always \
    --network "${NETWORK_NAME}" \
    -p 8080:8080 \
    -e MYSQL_DSN="lynxgate:${DB_PASSWORD}@tcp(${DB_CONTAINER}:3306)/lynxgate" \
    -e MYSQL_ENCRYPTION_KEY="7AE49A19B3C844BDB68E460D9224A5D0" \
    -e CURRENT_DATE="2024-04-01" \
    "${APP_CONTAINER}"

# Wait for app to be ready again
RETRY_COUNT=0
while ! curl -s "http://$APP_IP:8080/ready" > /dev/null 2>&1; do
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        fail "Timeout waiting for application to restart"
    fi
    echo "ℹ️ Waiting for app restart... ($(($RETRY_COUNT + 1))/$MAX_RETRIES)"
    sleep 2
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

# After container restart and ready check
echo "ℹ️ Waiting for quota reset to take effect..."
sleep 5

# Verify quota has reset
echo "ℹ️ Verifying quota has reset..."
RESPONSE=$(curl -s -w "%{http_code}" \
    -H "Authorization: $RESET_TOKEN" \
    "http://$APP_IP:8080/validate")

if [[ $RESPONSE != *"200"* ]]; then
    fail "Quota should have reset, expected 200, got: $RESPONSE"
fi

echo "✅ Quota reset working correctly"

# Clean up
echo "ℹ️ Cleaning up reset test token..."
curl -s -X DELETE "http://$APP_IP:8080/tokens/QuotaResetTest"

echo "✅ All tests passed successfully!"