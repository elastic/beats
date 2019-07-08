ARG TAG=9.5.3
FROM postgres:${TAG}
COPY docker-entrypoint-initdb.d /docker-entrypoint-initdb.d
HEALTHCHECK --interval=10s --retries=6 CMD psql -h localhost -U postgres -l
