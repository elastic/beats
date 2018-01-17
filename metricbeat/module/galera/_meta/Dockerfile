FROM centos:7

RUN yum update
RUN rpm -iv http://repo.mysql.com/yum/mysql-5.7-community/el/7/x86_64/mysql57-community-release-el7-7.noarch.rp
RUN rpm -iv http://releases.galeracluster.com/galera-3/centos/7/x86_64
RUN rpm -iv http://releases.galeracluster.com/mysql-wsrep-5.7/centos/7/x86_64

RUN yum install -y \
    mysql-community-client \
    galera-3 \
    mysql-wsrep-5.7 \
    netcat

RUN mysqld --defaults-extra-file=/var/lib/mysql/my.cnf & \ 
    until [ -S /var/lib/mysql/mysql.sock ]; do sleep 1; done && \
    for i in $(ls /tmp/*.sql); do echo dumping $i && mysql -u root < $i; done && \
    mysql -u root -e "CREATE USER 'mysql'@'%' IDENTIFIED BY 'mysql';" && \
    mysql -u root -e "GRANT ALL ON *.* TO 'mysql'@'%';" && \
    mysqladmin -u root -p shutdown

COPY # TODO copy assets

HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 3360

# ENV MYQSL_ROOT_PASSWORD test

CMD [ "/usr/sbin/mysqld", "--defaults-extra-file=/var/lib/mysql/my.cnf" ]