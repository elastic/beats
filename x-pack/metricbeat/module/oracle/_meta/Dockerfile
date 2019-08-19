FROM store/oracle/database-enterprise:12.2.0.1

HEALTHCHECK --interval=1s --retries=90 CMD /usr/bin/echo 'select 1' | sqlplus sys/Oradoc_db1@localhost:1521/ORCLPDB1.localdomain AS SYSDBA
