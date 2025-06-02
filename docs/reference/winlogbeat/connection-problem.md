---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/connection-problem.html
---

# Logstash connection doesn't work [connection-problem]

You may have configured {{ls}} or Winlogbeat incorrectly. To resolve the issue:

* Make sure that {{ls}} is running and you can connect to it. First, try to ping the {{ls}} host to verify that you can reach it from the host running Winlogbeat. Then use either `nc` or `telnet` to make sure that the port is available. For example:

    ```shell
    ping <hostname or IP>
    telnet <hostname or IP> 5044
    ```

* Verify that the config file for Winlogbeat specifies the correct port where {{ls}} is running.
* Make sure that the {{es}} output is commented out in the config file and the {{ls}} output is uncommented.
* Confirm that the most recent [Beats input plugin for {{ls}}](logstash-docs-md://lsr/plugins-inputs-beats.md) is installed and configured. Note that Beats will not connect to the Lumberjack input plugin. To learn how to install and update plugins, see [Working with plugins](logstash://reference/working-with-plugins.md).

