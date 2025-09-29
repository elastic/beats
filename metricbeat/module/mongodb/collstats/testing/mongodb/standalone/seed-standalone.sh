#!/usr/bin/env bash
set -euo pipefail

echo "[DEBUG] Starting seed-standalone.sh script"
echo "[DEBUG] Environment variables:"
echo "[DEBUG]   COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-<not set>}"
echo "[DEBUG]   MONGO_PORT=${MONGO_PORT:-<not set>}"
echo "[DEBUG]   PWD=$(pwd)"

# Detect Docker Compose project prefix - defaults to current directory name
if [ -z "${COMPOSE_PROJECT_NAME:-}" ]; then
    PROJECT_PREFIX=$(basename "$(pwd)")
    echo "[DEBUG] Using directory name for PROJECT_PREFIX: $PROJECT_PREFIX"
else
    PROJECT_PREFIX="$COMPOSE_PROJECT_NAME"
    echo "[DEBUG] Using COMPOSE_PROJECT_NAME for PROJECT_PREFIX: $PROJECT_PREFIX"
fi

# Try both naming conventions (Docker Compose v1 uses underscores, v2 uses hyphens)
CONTAINER_NAME_HYPHEN="${PROJECT_PREFIX}-mongodb-1"
CONTAINER_NAME_UNDERSCORE="${PROJECT_PREFIX}_mongodb_1"

echo "[DEBUG] Looking for container with names:"
echo "[DEBUG]   Hyphen format: ${CONTAINER_NAME_HYPHEN}"
echo "[DEBUG]   Underscore format: ${CONTAINER_NAME_UNDERSCORE}"

# List all containers for debugging
echo "[DEBUG] Current Docker containers:"
docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | head -20

# Check which container name exists
if docker ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME_HYPHEN}$"; then
    CONTAINER_NAME="${CONTAINER_NAME_HYPHEN}"
    echo "[DEBUG] Found container with hyphen format: ${CONTAINER_NAME}"
elif docker ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME_UNDERSCORE}$"; then
    CONTAINER_NAME="${CONTAINER_NAME_UNDERSCORE}"
    echo "[DEBUG] Found container with underscore format: ${CONTAINER_NAME}"
else
    echo "[ERROR] Could not find container with name ${CONTAINER_NAME_HYPHEN} or ${CONTAINER_NAME_UNDERSCORE}"
    echo "[ERROR] Available containers:"
    docker ps -a --format "{{.Names}}" | grep -i "${PROJECT_PREFIX}" || echo "[ERROR] No containers matching project prefix: ${PROJECT_PREFIX}"
    exit 1
fi

DB_NAME="${DB_NAME:-mbtest}"

retry() {
  local n=0
  until "$@"; do
    n=$((n+1))
    echo "[DEBUG] Retry attempt $n/30 for command: $*"
    if [ $n -ge 30 ]; then
      echo "[ERROR] Command failed after 30 attempts: $*"
      return 1
    fi
    sleep 2
  done
  echo "[DEBUG] Command succeeded on attempt $n: $*"
}

shell_cmd() {
  local cn=$1; shift
  echo "[DEBUG] Detecting shell command for container: $cn" >&2
  # Try mongosh first (MongoDB 5.0+)
  if docker exec "$cn" which mongosh >/dev/null 2>&1; then
    echo "[DEBUG] Found mongosh in container" >&2
    echo "mongosh"
    return
  fi
  # Fall back to mongo for older versions
  if docker exec "$cn" which mongo >/dev/null 2>&1; then
    echo "[DEBUG] Found mongo in container" >&2
    echo "mongo"
    return
  fi
  echo "[ERROR] Neither mongosh nor mongo found in container" >&2
  return 1
}

echo "[INFO] Seeding MongoDB standalone instance"
echo "[INFO] Container: $CONTAINER_NAME"
echo "[INFO] Database: $DB_NAME"

# Detect shell command
echo "[DEBUG] Detecting MongoDB shell command..."
if ! SHELL_CMD=$(shell_cmd "$CONTAINER_NAME"); then
    echo "[ERROR] Failed to detect MongoDB shell command"
    docker exec "$CONTAINER_NAME" ls -la /usr/bin/ | grep -E "mongo|mongosh" || true
    exit 1
fi
echo "[INFO] Using shell: $SHELL_CMD"

# Wait for MongoDB
echo "[INFO] Waiting for MongoDB to be ready..."
echo "[DEBUG] Testing connection with: docker exec $CONTAINER_NAME $SHELL_CMD --quiet --eval \"db.adminCommand('ping')\""
if ! retry docker exec "$CONTAINER_NAME" $SHELL_CMD --quiet --eval "db.adminCommand('ping')"; then
    echo "[ERROR] MongoDB failed to become ready after 30 attempts"
    echo "[DEBUG] Container logs:"
    docker logs "$CONTAINER_NAME" --tail=50
    exit 1
fi
echo "[INFO] MongoDB is ready"

# Seed database
echo "[INFO] Seeding database '$DB_NAME'..."
echo "[DEBUG] Running seed commands via: docker exec -i $CONTAINER_NAME $SHELL_CMD --quiet"
cat <<'JS' | docker exec -i "$CONTAINER_NAME" $SHELL_CMD --quiet
use mbtest

// Drop existing collections
db.getCollectionNames().forEach(function(c) {
    if (!c.startsWith('system.')) {
        db.getCollection(c).drop();
    }
});

// Create main collection
db.createCollection("test_collection");
for (let i = 1; i <= 10000; i++) {
    db.test_collection.insert({
        _id: i,
        userId: Math.floor(Math.random() * 1000),
        username: "user_" + (i % 100),
        timestamp: new Date(Date.now() - Math.random() * 86400000 * 365),
        status: ['active', 'inactive', 'pending'][i % 3],
        data: {
            field1: 'x'.repeat((i % 128) + 1),
            field2: Math.random() * 1000,
            nested: {
                value: "nested_" + i
            }
        },
        tags: ["tag_" + (i % 20)]
    });
}

// Create indexed collection
db.createCollection("test_indexed");
db.test_indexed.createIndex({ userId: 1 });
db.test_indexed.createIndex({ timestamp: -1 });

for (let i = 1; i <= 5000; i++) {
    db.test_indexed.insert({
        _id: i,
        userId: i % 500,
        timestamp: new Date(),
        tags: ['tagA', 'tagB', 'tagC'].slice(0, (i % 3) + 1),
        status: ['active', 'inactive'][i % 2],
        score: Math.random() * 100
    });
}

// Create capped collection
db.createCollection("test_capped", {
    capped: true,
    size: 5242880,  // 5MB
    max: 5000
});

for (let i = 1; i <= 5000; i++) {
    db.test_capped.insert({
        event: "event_" + i,
        timestamp: new Date(),
        level: ['INFO', 'WARN', 'ERROR'][i % 3],
        message: "Log message " + i
    });
}

print("Database seeded successfully");
JS

echo "[INFO] Seeding completed successfully!"
echo "[INFO] MongoDB is available at: localhost:${MONGO_PORT:-27017}"
echo "[INFO] Database: $DB_NAME"
echo "[DEBUG] Script completed successfully"