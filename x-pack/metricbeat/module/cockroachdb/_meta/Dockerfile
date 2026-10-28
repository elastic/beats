ARG COCKROACHDB_VERSION
FROM cockroachdb/cockroach:v${COCKROACHDB_VERSION}

HEALTHCHECK --interval=1s --retries=90 CMD curl --fail -q http://localhost:8080/_status/vars

CMD ["start-single-node", "--insecure"]
