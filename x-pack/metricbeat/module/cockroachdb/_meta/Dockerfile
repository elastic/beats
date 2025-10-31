ARG COCKROACHDB_VERSION
FROM cockroachdb/cockroach:v${COCKROACHDB_VERSION}

HEALTHCHECK --interval=1s --retries=90 CMD curl -q http://localhost:8080/_stats/vars

CMD ["start-single-node", "--insecure"]
