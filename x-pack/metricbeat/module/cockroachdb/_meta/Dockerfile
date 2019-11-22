ARG COCKROACHDB_VERSION
FROM cockroachdb/cockroach:v${COCKROACHDB_VERSION}

RUN apt-get update && apt-get install -y curl

HEALTHCHECK --interval=1s --retries=90 CMD curl -q http://localhost:8080/_stats/vars

CMD ["start", "--insecure"]
