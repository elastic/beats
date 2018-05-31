FROM postgres:9.5.3
HEALTHCHECK --interval=10s --retries=6 CMD psql -h localhost -U postgres -l
COPY ./postgresql.conf /etc/postgresql/9.5/main/postgresql.conf