FROM oraclelinux:7
MAINTAINER Tamás Gulácsi <tgulacsi78@gmail.com>

RUN curl -o oracle-database-preinstall-18c-1.0-1.el7.x86_64.rpm https://yum.oracle.com/repo/OracleLinux/OL7/latest/x86_64/getPackage/oracle-database-preinstall-18c-1.0-1.el7.x86_64.rpm
RUN yum -y localinstall oracle-database-preinstall-18c-1.0-1.el7.x86_64.rpm

# Download oracle-database-xe-18c-1.0-1.x86_64.rpm from
# https://www.oracle.com/technetwork/database/database-technologies/express-edition/downloads/index.html
RUN echo 'Download oracle-database-xe-18c-1.0-1.x86_64.rpm from https://www.oracle.com/technetwork/database/database-technologies/express-edition/downloads/index.html'
COPY oracle-database-xe-18c-1.0-1.x86_64.rpm .
ENV ORACLE_DOCKER_INSTALL=true
RUN yum -y localinstall oracle-database-xe-18c-1.0-1.x86_64.rpm

RUN rm oracle-database-xe-18c-1.0-1.x86_64.rpm
ARG ORACLE_PASSWORD=test
RUN env ORACLE_PASSWORD=$ORACLE_PASSWORD ORACLE_CONFIRM_PASSWORD=$ORACLE_PASSWORD /etc/init.d/oracle-xe-18c configure

EXPOSE 1521/tcp 5500/tcp
ENTRYPOINT ["/bin/sh", "-c", "/etc/init.d/oracle-xe-18c start; sleep 3600"]
