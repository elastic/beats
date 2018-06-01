FROM postgres:9.5.3
COPY docker-entrypoint-initdb.d /docker-entrypoint-initdb.d
HEALTHCHECK --interval=10s --retries=6 CMD psql -h localhost -U postgres -l