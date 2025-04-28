---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-not-collecting-lines.html
---

# Filebeat isn't collecting lines from a file [filebeat-not-collecting-lines]

Filebeat might be incorrectly configured or unable to send events to the output. To resolve the issue:

* If using modules, make sure the `var.paths` setting points to the file. If configuring an input manually, make sure the `paths` setting is correct.
* Verify that the file is not older than the value specified by [`ignore_older`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-ignore-older). `ignore_older` is disable by default so this depends on the value you have set. You can change this behavior by specifying a different value for [`ignore_older`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-ignore-older).
* Make sure that Filebeat is able to send events to the configured output. Run Filebeat in debug mode to determine whether it’s publishing events successfully:

    ```sh
    ./filebeat -c config.yml -e -d "*"
    ```


