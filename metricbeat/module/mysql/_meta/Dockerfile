ARG MYSQL_IMAGE=mysql:5.7.12
FROM $MYSQL_IMAGE

ENV MYSQL_ROOT_PASSWORD test

HEALTHCHECK --interval=1s --retries=90 CMD mysql -u root -p$MYSQL_ROOT_PASSWORD -P 3306 -e "SELECT 1"

COPY test.cnf /etc/mysql/conf.d/test.cnf
