---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/migrate-to-filestream.html
---

# Migrate log input configurations to filestream [migrate-to-filestream]

The `filestream` input has been generally available since 7.14 and it is highly recommended you migrate your existing `log` input configurations. The `filestream` input comes with many improvements over the old `log` input, such as configurable order for parsers and more.

The `log` input is deprecated and will eventually be removed from Filebeat. We are not fixing new issues or adding any enhancements to the `log` input. Our focus is on `filestream`.

This manual migration is required only if you’ve defined `log` inputs manually in your stand-alone Filebeat configuration. All the integrations or modules that are still using `log` inputs under the hood will be eventually migrated automatically without any additional actions required from the user.

In this guide, you’ll learn how to migrate an existing `log` input configuration.

::::{important}
You must replace `log` inputs with `filestream` inputs, make sure you have removed all the old `log` inputs from the configuration before starting Filebeat with the new `filestream` inputs. Running old `log` inputs and new `filestream` inputs pointed to the same files will lead to data duplication.
::::


The following example shows three `log` inputs:

```yaml
filebeat.inputs:
 - type: log
   enabled: true
   paths:
     - /var/log/java-exceptions*.log
   multiline:
    pattern: '^\['
    negate: true
    match: after
  close_removed: true
  close_renamed: true

- type: log
  enabled: true
  paths:
    - /var/log/my-application*.json
  scan_frequency: 1m
  json.keys_under_root: true

- type: log
  enabled: true
  paths:
    - /var/log/my-old-files*.log
  tail_files: true
```

For this example, let’s assume that the `log` input is used to collect logs from the following files. The progress of data collection is shown for each file.

```sh
/var/log/java-exceptions1.log (100%)
/var/log/java-exceptions2.log (100%)
/var/log/java-exceptions3.log (75%)
/var/log/java-exceptions4.log (0%)
/var/log/java-exceptions5.log (0%)
/var/log/my-application1.json (100%)
/var/log/my-application2.json (5%)
/var/log/my-application3.json (0%)
/var/log/my-old-files1.json (0%)
```







