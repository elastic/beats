ARG AEROSPIKE_VERSION
FROM aerospike/aerospike-server-enterprise:${AEROSPIKE_VERSION}

HEALTHCHECK --interval=1s --retries=90 CMD asinfo -v "namespace/test" | grep "ns_cluster_size=1" -q
