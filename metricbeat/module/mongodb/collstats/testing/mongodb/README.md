# MongoDB Testing Setups

This directory provides minimal Docker Compose setups to run MongoDB locally for testing.

Available setups:
- `standalone/` – single MongoDB instance
- `replica_set/` – a MongoDB replica set
- `sharded/` – a sharded MongoDB cluster

## Prerequisites
- Docker and Docker Compose installed and running
- Bash shell (macOS/Linux)

## Choose MongoDB version
These setups use Docker images like `${MONGO_IMAGE:-mongo}:${MONGO_VERSION:-7.0}`. If you don’t set `MONGO_VERSION`, it defaults to `7.0`.

To run a specific MongoDB version, prefix your compose command with the variable (examples for zsh):

```bash
# Use MongoDB 5.0
MONGO_VERSION=5.0 docker-compose up -d

# Or with Docker Compose V2 syntax
MONGO_VERSION=7.0 docker compose up -d
```

You can also override the image name if needed:

```bash
MONGO_IMAGE=mongo MONGO_VERSION=7.0 docker-compose up -d
```

## Quick start
Run each from its own subdirectory.

### Standalone
```bash
cd standalone
docker-compose up -d
./seed-standalone.sh
```

### Replica set
```bash
cd replica_set
docker-compose up -d
./init-replica-set.sh
```

### Sharded cluster
```bash
cd sharded
docker-compose up -d
./init-sharded-cluster.sh
./verify-sharding.sh
```

## Stop and clean up
From within a subdirectory:
```bash
docker-compose down -v
```

These environments are for local testing only.