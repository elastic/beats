# MongoDB test matrix for collstats

This folder provides Docker Compose setups to test the collstats metricset across MongoDB versions and topologies:

- Standalone: basic sanity checks
- Sharded: validate fields that must be summed/averaged/merged across shards

Notes on versions:
- Official Docker tags exist for 5.0, 6.0, 7.0 in the `mongo` image. A 6.2 tag may not be present in the official image. For the 6.2+ path ($collStats) you can use 7.0 which behaves equivalently for our purposes. If you insist on 6.2, try `mongodb/mongodb-community-server` images if available, or test with 7.0 instead.

## Quick start (standalone)

1) Choose image and version via env vars:

- MONGO_IMAGE (default: mongo)
- MONGO_VERSION (default: 7.0)

2) Bring up a standalone node:

```zsh
cd module/mongodb/collstats/testing/mongodb
MONGO_VERSION=5.0 docker compose -f compose.standalone.yml up -d
# or for 7.0
MONGO_VERSION=7.0 docker compose -f compose.standalone.yml up -d
```

3) Seed a few collections:

```zsh
./seed-standalone.sh
```

4) Point Metricbeat to mongodb://localhost:27017 in your module config and run Metricbeat from the repo root.

5) Tear down:

```zsh
docker compose -f compose.standalone.yml down -v
```

## Sharded cluster (end-to-end collstats merge testing)

This brings up a minimal sharded cluster: 3x config servers, 2 shard replica sets (1-node each for simplicity), and a mongos router. It then initializes the replsets, adds shards, enables sharding on `mbtest`, shards two collections (`coll_hash`, `coll_range`), pre-splits and moves chunks, and seeds documents.

1) Start the cluster:

```zsh
# 7.0 (covers the 6.2+ $collStats path)
MONGO_VERSION=7.0 docker compose -f compose.sharded.yml up -d

# Optional: try 5.0 to validate legacy collStats command path
MONGO_VERSION=5.0 docker compose -f compose.sharded.yml up -d
```

2) Initialize and seed:

```zsh
./init-sharded.sh
```

3) Verify sharding is active and chunks placed on both shards:

```zsh
./verify-sharding.sh
```

You should see chunks for `mbtest.coll_range` split at userId 0, distributed across shard01 and shard02. The hashed collection will distribute automatically as you insert more docs.

4) Run Metricbeat against mongos at mongodb://localhost:27017 and confirm events reflect merged values (e.g., `count`, `size`, `totalIndexSize`, `indexSizes` sums, `avgObjSize` weighted average). Check debug logs added in collstats for method selection and merge summary.

5) Tear down:

```zsh
docker compose -f compose.sharded.yml down -v
```

## Environment variables

- MONGO_IMAGE: default `mongo`
- MONGO_VERSION: default `7.0`

If `mongo:6.2` isn’t available, use `MONGO_IMAGE=mongodb/mongodb-community-server` with an appropriate version that supports $collStats (e.g., `7.0`).

## Troubleshooting

- **Container naming**: Docker Compose adds a project prefix to container names (default: directory name, e.g., `mongodb-config1-1`). The updated `init-sharded.sh` script automatically detects and uses the correct container names with the project prefix.
- **Replica set initialization errors**: If you see "FailedToSatisfyReadPreference" or "no replset config has been received" errors, it means the replica sets haven't been initialized. Run `./init-sharded.sh` to properly initialize all replica sets with the correct hostnames.
- **Custom project names**: If using a custom Docker Compose project name via `COMPOSE_PROJECT_NAME` or `-p` flag, set the same value when running init scripts: `COMPOSE_PROJECT_NAME=myproject ./init-sharded.sh`
- If mongosh isn't found in the image, the scripts will try `mongo` shell instead.
- Init can take a few seconds; scripts include small waits and retries.
- On Apple Silicon: Docker will pull the `-arm64` variants automatically if available.
