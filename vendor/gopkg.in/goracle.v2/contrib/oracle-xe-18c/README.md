# Oracle XE 18c
Be patient. The rpm below is 2.5GB, the installation takes anywhere from half an hour to several hours to complete!

But at the end, you'll have an Oracle DB running inside a container,
with a container database (XE) with one pluggable database (XEPDB1),
with the listener at the default port (1521) and Enterprise Manager Express on port 5500.

## Build

  1. Download oracle-database-xe-18c-1.0-1.x86_64.rpm from https://www.oracle.com/technetwork/database/database-technologies/express-edition/downloads/index.html
  2. `docker build --build-arg ORACLE_PASSWORD=mySecr4tPassw0rd -t my/oracle18c:latest .`

## Use

	docker run --init --rm --name oracle my/oracle18c:latest
