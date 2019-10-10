ARG POSTGRESQL_VERSION
FROM postgres:${POSTGRESQL_VERSION}
COPY docker-entrypoint-initdb.d /docker-entrypoint-initdb.d
HEALTHCHECK --interval=10s --retries=6 CMD psql -h localhost -U postgres -l
