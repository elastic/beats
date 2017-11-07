FROM mysql:5.7.12
RUN apt-get update && apt-get install -y netcat
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 3306

ENV MYSQL_ROOT_PASSWORD test
