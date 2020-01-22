WHENEVER SQLERROR CONTINUE

DROP USER test CASCADE;

WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK

CREATE USER test IDENTIFIED BY r97oUPimsmTOIcBaeeDF;
ALTER USER test QUOTA 100m ON data;
GRANT create session, create table, create type, create sequence, create synonym, create procedure, change notification TO test;
GRANT EXECUTE ON SYS.DBMS_AQ TO test;
GRANT EXECUTE ON SYS.DBMS_AQADM TO test;

GRANT create user, drop user, alter user TO test;
GRANT connect TO test WITH admin option;
GRANT create session TO test WITH admin option;
