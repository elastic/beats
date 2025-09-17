#!/usr/bin/env bash
set -euo pipefail

mongo_shell() {
  if command -v mongosh >/dev/null 2>&1; then
    echo mongosh
  else
    echo mongo
  fi
}

SHELL_BIN=$(mongo_shell)

# Wait for healthy
for i in {1..60}; do
  if docker inspect --format='{{.State.Health.Status}}' collstats-mongo-standalone 2>/dev/null | grep -q healthy; then
    break
  fi
  sleep 2
  if [ $i -eq 60 ]; then
    echo "Mongo did not become healthy in time" >&2
    exit 1
  fi
done

echo "Seeding standalone"
cat <<'JS' | docker exec -i collstats-mongo-standalone ${SHELL_BIN} --quiet --username root --password example --authenticationDatabase admin
use mbtest

// Create and populate a few collections
for (let i = 0; i < 3; i++) {
  const name = `coll_${i}`
  db[name].drop()
  let bulk = []
  for (let j = 0; j < 5000; j++) {
    bulk.push({ _id: j, userId: j % 100, payload: 'x'.repeat((j % 256) + 1) })
    if (bulk.length === 1000) {
      db[name].insertMany(bulk)
      bulk = []
    }
  }
  if (bulk.length) db[name].insertMany(bulk)
  db[name].createIndex({ userId: 1 })
}

print('Standalone seed complete')
JS

echo "Done."
