---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-oracle.html
---

# Oracle module [metricbeat-module-oracle]

This is the [Oracle](https://www.oracle.com) module for Metricbeat. It is under active development with feedback from the community. A single Metricset for Tablespace monitoring is added so the community can start gathering metrics from their nodes and contributing to the module.


## Compatibility [_compatibility_41]

Oracle Metricbeat module is being tested version 12c R2 by using the store/oracle/database-enterprise:12.2.0.1 Docker image downloaded directly from the [Oracle Docker Hub repository](https://hub.docker.com/_/oracle-database-enterprise-edition) which is based on 5.0.13-arch1-1-ARCH Arch Linux. This is important, the module has only been tested with the mentioned image in Linux environments.


## Dashboard [_dashboard_36]

An overview dashboard for Kibana is already included:

![metricbeat oracle overview](images/metricbeat-oracle-overview.png)


## Requirements [_requirements]

Connectivity to Oracle can be facilitated in two ways either by using official Oracle libraries or by using a JDBC driver. Facilitation of the connectivity using JDBC is not supported currently with Metricbeat. Connectivity can be facilitated using Oracle libraries and the detailed steps to do the same are mentioned below.

**Oracle Database Connection Pre-requisites**

To get connected with the Oracle Database ORACLE_SID, ORACLE_BASE, ORACLE_HOME environment variables should be set.

For example: Let’s consider Oracle Database 21c installation using RPM manually by following [this](https://docs.oracle.com/en/database/oracle/oracle-database/21/ladbi/running-rpm-packages-to-install-oracle-database.html) link, environment variables should be set as follows: `ORACLE_SID=ORCLCDB` `ORACLE_BASE=/opt/oracle/oradata` `ORACLE_HOME=/opt/oracle/product/21c/dbhome_1` Also, add `ORACLE_HOME/bin` to the `PATH` environment variable.

**Oracle Instant Client**

Oracle Instant Client enables development and deployment of applications that connect to Oracle Database. The Instant Client libraries provide the necessary network connectivity and advanced data features to make full use of Oracle Database. If you have OCI Oracle server which comes with these libraries pre-installed, you don’t need a separate client installation.

The OCI library install few Client Shared Libraries that must be referenced on the machine where Metricbeat is installed. Please follow [this](https://docs.oracle.com/en/database/oracle/oracle-database/21/lacli/install-instant-client-using-zip.html#GUID-D3DCB4FB-D3CA-4C25-BE48-3A1FB5A22E84) link for OCI Instant Client set up. The OCI Instant Client is available with the Oracle Universal Installer, RPM file or ZIP file. Download links can be found [here](https://www.oracle.com/database/technologies/instant-client/downloads.html).

**Enable Listener**

The Oracle listener is a service that runs on the database host and receives requests from Oracle clients. Make sure that [listener](https://docs.oracle.com/cd/B19306_01/network.102/b14213/lsnrctl.htm) should be running. To check if the listener is running or not, run:

`lsnrctl STATUS`

If the listener is not running, use the command to start:

`lsnrctl START`

Then, Metricbeat can be launched.

**Host Configuration**

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


## Metricsets [_metricsets_58]

The following Metricsets are included in the module:


### `performance` [_performance_2]

Includes performance related events which contains mainly cursor and cache based data.


### `tablespaces` [_tablespaces]

Includes information about data files and temp files, grouped by Tablespace with free space available, used space, status of the data files, status of the Tablespace, etc.


### `sysmetric` [_sysmetric]

Includes the system metric values captured for the most current time interval from Oracle system metrics.


### Example configuration [_example_configuration_51]

The Oracle module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
# Module: oracle

- module: oracle
  period: 10m
  metricsets:
    - tablespace
  enabled: true
  hosts: ['user="user" password="pass" connectString="0.0.0.0:1521/ORCLPDB1.localdomain"']
- module: oracle
  period: 10s
  metricsets:
    - performance
  enabled: true
  hosts: ['user="user" password="pass" connectString="0.0.0.0:1521/ORCLPDB1.localdomain"']
- module: oracle
  period: 60s
  metricsets:
    - sysmetric
  enabled: true
  hosts: ['user="user" password="pass" connectString="0.0.0.0:1521/ORCLPDB1.localdomain"']
  # patterns: ["foo%","%bar","%foobar%"]

  # username: ""
  # password: ""
```


### Metricsets [_metricsets_59]

The following metricsets are available:

* [performance](/reference/metricbeat/metricbeat-metricset-oracle-performance.md)
* [sysmetric](/reference/metricbeat/metricbeat-metricset-oracle-sysmetric.md)
* [tablespace](/reference/metricbeat/metricbeat-metricset-oracle-tablespace.md)




