ARG AEROSPIKE_VERSION
FROM aerospike/aerospike-server-enterprise:${AEROSPIKE_VERSION}

HEALTHCHECK --interval=1s --retries=90 CMD asinfo -v namespaces | grep -i -q "test" || exit 1
