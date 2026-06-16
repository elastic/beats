FROM docker.elastic.co/observability-ci/database-enterprise:12.2.0.1

ENV ORACLE_BASE=/db/oracle
ENV ORACLE_PASSWORD=Oradoc_db1

HEALTHCHECK --interval=30s --timeout=15s --start-period=60s --retries=5 CMD "bash -c echo 'select sysdate from dual;' | ORACLE_HOME=/u01/app/oracle/product/12.2.0/dbhome_1/ /u01/app/oracle/product/12.2.0/dbhome_1/bin/sqlplus -s sys/Oradoc_db1@oracle:1521/ORCLCDB.localdomain as sysdba"