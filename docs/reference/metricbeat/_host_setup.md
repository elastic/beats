---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/_host_setup.html
---

# Host Setup [_host_setup]

Some drivers require additional configuration to work. Find here instructions for these drivers.

## Oracle Database Connection Pre-requisites [_oracle_database_connection_pre_requisites]

To get connected with the Oracle Database `ORACLE_SID`, `ORACLE_BASE`, `ORACLE_HOME` environment variables should be set.

For example: Let us consider Oracle Database 21c installation using RPM manually by following [this](https://docs.oracle.com/en/database/oracle/oracle-database/21/ladbi/running-rpm-packages-to-install-oracle-database.html) link, environment variables should be set as follows:

```bash
export ORACLE_BASE=/opt/oracle/oradata
export ORACLE_HOME=/opt/oracle/product/21c/dbhome_1
```

Also, add `ORACLE_HOME/bin` to the `PATH` environment variable.

### Oracle Instant Client Installation [_oracle_instant_client_installation]

Oracle Instant Client enables the development and deployment of applications that connect to the Oracle Database. The Instant Client libraries provide the necessary network connectivity and advanced data features to make full use of the Oracle Database. If you have an OCI Oracle server which comes with these libraries pre-installed, you don’t need a separate client installation.

The OCI library installs a few Client Shared Libraries that must be referenced on the machine where Metricbeat is installed. Please follow [this](https://docs.oracle.com/en/database/oracle/oracle-database/21/lacli/install-instant-client-using-zip.html#GUID-D3DCB4FB-D3CA-4C25-BE48-3A1FB5A22E84) link for OCI Instant Client set up. The OCI Instant Client is available with the Oracle Universal Installer, RPM file or ZIP file. Download links can be found at [here](https://www.oracle.com/database/technologies/instant-client/downloads.html).


### Enable Oracle Listener [_enable_oracle_listener]

The Oracle listener is a service that runs on the database host and receives requests from Oracle clients. Make sure that [listener](https://docs.oracle.com/cd/B19306_01/network.102/b14213/lsnrctl.htm) should be running. To check if the listener is running or not, run:

```bash
lsnrctl STATUS
```

If the listener is not running, use the command to start:

```bash
lsnrctl START
```

Then, Metricbeat can be launched.


### Host Configuration for Oracle [_host_configuration_for_oracle]

The following types of host configuration are supported:

1. An old-style Oracle connection string, for backwards compatibility:

    1. `hosts: ["user/pass@0.0.0.0:1521/ORCLPDB1.localdomain"]`
    2. `hosts: ["user/password@0.0.0.0:1521/ORCLPDB1.localdomain as sysdba"]`

2. DSN configuration as a URL:

    1. `hosts: ["oracle://user:pass@0.0.0.0:1521/ORCLPDB1.localdomain?sysdba=1"]`

3. DSN configuration as a logfmt-encoded parameter list:

    1. `hosts: ['user="user" password="pass" connectString="0.0.0.0:1521/ORCLPDB1.localdomain"']`
    2. `hosts: ['user="user" password="password" connectString="host:port/service_name" sysdba=true']`


DSN host configuration is the recommended configuration type as it supports the use of special characters in the password.

In a URL any special characters should be URL encoded.

In the logfmt-encoded DSN format, if the password contains a backslash character (`\`), it must be escaped with another backslash. For example, if the password is `my\_password`, it must be written as `my\\_password`.

The username and password to connect to the database can be provided as values to the `username` and `password` keys of `sql.yml`.

```yaml
- module: sql
  metricsets:
    - query
  period: 10s
  driver: "oracle"
  enabled: true
  hosts: ['user="" password="" connectString="0.0.0.0:1521/ORCLCDB.localdomain" sysdba=true']
  username: sys
  password: password
  sql_queries:
  - query: SELECT METRIC_NAME, VALUE FROM V$SYSMETRIC WHERE GROUP_ID = 2 and METRIC_NAME LIKE '%'
    response_format: variables
```


## Example configuration [_example_configuration_59]

The SQL module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: sql
  metricsets:
    - query
  period: 10s
  hosts: ["user=myuser password=mypassword dbname=mydb sslmode=disable"]

  driver: "postgres"
  sql_query: "select now()"
  sql_response_format: table
```


## Metricsets [_metricsets_68]

The following metricsets are available:

* [query](/reference/metricbeat/metricbeat-metricset-sql-query.md)



