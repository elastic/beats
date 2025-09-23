#!/usr/bin/env bash
set -euo pipefail

# Detect Docker Compose project prefix - defaults to current directory name
if [ -z "${COMPOSE_PROJECT_NAME:-}" ]; then
    PROJECT_PREFIX=$(basename "$(pwd)")
else
    PROJECT_PREFIX="$COMPOSE_PROJECT_NAME"
fi
CONTAINER_NAME="${PROJECT_PREFIX}-mongodb-1"
DB_NAME="${DB_NAME:-mbtest}"

retry() {
  local n=0
  until "$@"; do
    n=$((n+1))
    if [ $n -ge 30 ]; then
      return 1
    fi
    sleep 2
  done
}

shell_cmd() {
  local cn=$1; shift
  # Try mongosh first (MongoDB 5.0+)
  if docker exec "$cn" which mongosh >/dev/null 2>&1; then
    echo "mongosh"
    return
  fi
  # Fall back to mongo for older versions
  echo "mongo"
}

echo "Seeding MongoDB standalone instance"
echo "Container: $CONTAINER_NAME"
echo "Database: $DB_NAME"

# Detect shell command
SHELL_CMD=$(shell_cmd "$CONTAINER_NAME")
echo "Using shell: $SHELL_CMD"

# Wait for MongoDB
echo "Waiting for MongoDB to be ready..."
retry docker exec "$CONTAINER_NAME" $SHELL_CMD --quiet --eval "db.adminCommand('ping')"

# Seed database
echo "Seeding database '$DB_NAME'..."
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

echo "Seeding completed successfully!"
echo "MongoDB is available at: localhost:27017"
echo "Database: $DB_NAME"