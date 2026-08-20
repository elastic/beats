#!/usr/bin/env bash
set -euo pipefail

COLLSTATS_VERBOSE=0
case "${METRICBEAT_COLLSTATS_LOGS:-}" in
    1|true|TRUE|yes|YES)
        COLLSTATS_VERBOSE=1
        ;;
esac

if [ -z "${METRICBEAT_COLLSTATS_LOGS:-}" ] && [ -z "${CI:-}" ]; then
    COLLSTATS_VERBOSE=1
fi

log_info() {
    printf '[INFO] %s\n' "$*"
}

log_debug() {
    if [ "$COLLSTATS_VERBOSE" = "1" ]; then
        printf '[DEBUG] %s\n' "$*" >&2
    fi
}

log_debug "Starting seed-standalone.sh script"
log_debug "Shell: ${SHELL:-<unknown>}"
log_debug "Script: $0"
log_debug "Environment variables:"
log_debug "  COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-<not set>}"
log_debug "  MONGO_PORT=${MONGO_PORT:-<not set>}"
log_debug "  PWD=$(pwd)"
log_debug "  PATH=$PATH"

# Check if docker is available
if ! command -v docker >/dev/null 2>&1; then
    echo "[ERROR] docker command not found in PATH"
    exit 1
fi

# Detect Docker Compose project prefix - defaults to current directory name
if [ -z "${COMPOSE_PROJECT_NAME:-}" ]; then
    PROJECT_PREFIX=$(basename "$(pwd)")
    log_debug "Using directory name for PROJECT_PREFIX: $PROJECT_PREFIX"
else
    PROJECT_PREFIX="$COMPOSE_PROJECT_NAME"
    log_debug "Using COMPOSE_PROJECT_NAME for PROJECT_PREFIX: $PROJECT_PREFIX"
fi

# Try both naming conventions (Docker Compose v1 uses underscores, v2 uses hyphens)
CONTAINER_NAME_HYPHEN="${PROJECT_PREFIX}-mongodb-1"
CONTAINER_NAME_UNDERSCORE="${PROJECT_PREFIX}_mongodb_1"

log_debug "Looking for container with names:"
log_debug "  Hyphen format: ${CONTAINER_NAME_HYPHEN}"
log_debug "  Underscore format: ${CONTAINER_NAME_UNDERSCORE}"

if [ "$COLLSTATS_VERBOSE" = "1" ]; then
  log_debug "Current Docker containers:"
  docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null | head -20 || true
fi

# Check which container name exists (try underscore first as it's more common in CI)
if docker ps -a --format "{{.Names}}" 2>/dev/null | grep -q "^${CONTAINER_NAME_UNDERSCORE}$"; then
    CONTAINER_NAME="${CONTAINER_NAME_UNDERSCORE}"
    log_debug "Found container with underscore format: ${CONTAINER_NAME}"
elif docker ps -a --format "{{.Names}}" 2>/dev/null | grep -q "^${CONTAINER_NAME_HYPHEN}$"; then
    CONTAINER_NAME="${CONTAINER_NAME_HYPHEN}"
    log_debug "Found container with hyphen format: ${CONTAINER_NAME}"
else
    echo "[ERROR] Could not find container with name ${CONTAINER_NAME_HYPHEN} or ${CONTAINER_NAME_UNDERSCORE}"
    echo "[ERROR] Available containers:"
    docker ps -a --format "{{.Names}}" 2>/dev/null | grep -i "${PROJECT_PREFIX}" || echo "[ERROR] No containers matching project prefix: ${PROJECT_PREFIX}"
    exit 1
fi

DB_NAME="${DB_NAME:-mbtest}"

retry() {
  local n=0
  until "$@" 2>/dev/null; do
    n=$((n+1))
        log_debug "Retry attempt $n/30 for command: $*"
    if [ $n -ge 30 ]; then
      echo "[ERROR] Command failed after 30 attempts: $*" >&2
      return 1
    fi
    sleep 2
  done
    log_debug "Command succeeded on attempt $n: $*"
}

shell_cmd() {
  local cn=$1; shift
    log_debug "Detecting shell command for container: $cn"
  # Try mongosh first (MongoDB 5.0+)
  if docker exec "$cn" which mongosh >/dev/null 2>&1; then
        log_debug "Found mongosh in container"
    echo "mongosh"
    return
  fi
  # Fall back to mongo for older versions
  if docker exec "$cn" which mongo >/dev/null 2>&1; then
        log_debug "Found mongo in container"
    echo "mongo"
    return
  fi
    echo "[ERROR] Neither mongosh nor mongo found in container" >&2
  return 1
}

log_info "Seeding MongoDB standalone instance"
log_info "Container: $CONTAINER_NAME"
log_info "Database: $DB_NAME"

# Detect shell command
log_debug "Detecting MongoDB shell command..."
if ! SHELL_CMD=$(shell_cmd "$CONTAINER_NAME"); then
    echo "[ERROR] Failed to detect MongoDB shell command"
    docker exec "$CONTAINER_NAME" ls -la /usr/bin/ | grep -E "mongo|mongosh" || true
    exit 1
fi
log_info "Using shell: $SHELL_CMD"

# Wait for MongoDB
log_info "Waiting for MongoDB to be ready..."
log_debug "Testing connection with: docker exec $CONTAINER_NAME $SHELL_CMD --quiet --eval \"db.adminCommand('ping')\""
if ! retry docker exec "$CONTAINER_NAME" $SHELL_CMD --quiet --eval "db.adminCommand('ping')"; then
    echo "[ERROR] MongoDB failed to become ready after 30 attempts"
    echo "[DEBUG] Container logs:"
    docker logs "$CONTAINER_NAME" --tail=50
    exit 1
fi
log_info "MongoDB is ready"

# Seed database
log_info "Seeding database '$DB_NAME'..."
log_debug "Running seed commands via: docker exec -i $CONTAINER_NAME $SHELL_CMD --quiet"
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

log_info "Seeding completed successfully!"
log_info "MongoDB is available at: localhost:${MONGO_PORT:-27017}"
log_info "Database: $DB_NAME"
log_debug "Script completed successfully"