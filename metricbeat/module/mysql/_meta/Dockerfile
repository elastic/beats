ARG MYSQL_IMAGE
FROM ${MYSQL_IMAGE}

ENV MYSQL_ROOT_PASSWORD test

HEALTHCHECK --interval=1s --retries=90 CMD mysql -u root -p$MYSQL_ROOT_PASSWORD -h$HOSTNAME -P 3306 -e "SHOW STATUS" > /dev/null

COPY test.cnf /etc/mysql/conf.d/test.cnf
