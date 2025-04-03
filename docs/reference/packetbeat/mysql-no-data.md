---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/mysql-no-data.html
---

# Packetbeat isn���t capturing MySQL performance data [mysql-no-data]

You may be listening on the wrong interface or trying to capture data sent over an encrypted connection. Packetbeat can only monitor MySQL traffic if it is unencrypted. To resolve your issue:

* Make sure Packetbeat is configured to listen on the `lo0` interface:

    ```shell
    packetbeat.interfaces.device: lo0
    ```

* Make sure the client programs you are monitoring run `mysql` with SSL disabled. For example:

    ```shell
    mysql --protocol tcp --host=127.0.0.1 --port=3306 --ssl-mode=DISABLED
    ```


::::{important}
When SSL is disabled, the connection between the MySQL client and server is unencrypted, which means that anyone with access to your network may be able to inspect data sent between the client and server. If MySQL is running in an untrusted network, it’s not advisable to disable encryption.
::::


