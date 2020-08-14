FROM docker.elastic.co/observability-ci/database-enterprise:12.2.0.1

HEALTHCHECK --interval=1s --retries=90 CMD /usr/bin/echo 'select 1' | /u01/app/oracle/product/12.2.0/dbhome_1/bin/sqlplus sys/Oradoc_db1@localhost:1521/ORCLPDB1.localdomain AS SYSDBA
