---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/migrate-to-filestream.html
---

# Migrate `log` or `container` input configurations to filestream

::::{warning}
The [`container`](/reference/filebeat/filebeat-input-container.md) input is just a preset for the [`log`](/reference/filebeat/filebeat-input-log.md) input. The `log` input is deprecated in version 7.16 and it's disabled by default in version 9.0.

After deprecation it’s possible to use `log` or `container` input type (e.g. for this migration) only in combination with the `allow_deprecated_use: true` setting as a part of the input configuration.

The `log` and `container` input types will be eventually removed from Filebeat. We are not fixing new issues or adding any enhancements to the `log` or `container` inputs.
::::

The `filestream` input has been generally available since version 7.14 and it is highly recommended to migrate existing `log` input configurations to `filestream`. The `filestream` input comes with many improvements over the old `log` input, such as configurable order of parsers, better file identification, better scalability and more.

This manual migration is required only if you’ve defined `log` or `container` inputs manually in your standalone Filebeat configuration. All the integrations or modules that are still using `log` or `container` inputs under the hood will be eventually migrated automatically without any additional actions required from the user.

In this guide, you’ll learn how to migrate an existing `log` or `container` input configuration to `filestream`. This guide is also valid for using in the [autodiscover](/reference/filebeat/configuration-autodiscover.md) configuration.

::::{important}
Running old `log` or `container` inputs and new `filestream` inputs targeting the same files at the same time will cause data duplication. `log` and `container` inputs must be removed from the configuration once they are replaced with new `filestream` inputs.
::::

## Example configuration [example]

Let's say you have a Filebeat configuration file with the following `log` and `container` inputs:

```yaml
filebeat.inputs:
 - type: log
   enabled: true
   allow_deprecated_use: true
   paths:
     - /var/log/java-exceptions*.log
   multiline:
    pattern: '^\['
    negate: true
    match: after
  close_removed: true
  close_renamed: true

 - type: container
   allow_deprecated_use: true
   paths:
     - /var/lib/docker/containers/*.log

 - type: log
   enabled: true
   allow_deprecated_use: true
   paths:
     - /var/log/my-application*.json
   scan_frequency: 1m
   json.keys_under_root: true

 - type: log
   enabled: true
   allow_deprecated_use: true
   paths:
     - /var/log/my-old-files*.log
   tail_files: true
```

For this example, let’s assume that the `log` input is used to collect logs from the following files:

The percentage number indicates the data collection progress for each file.

```
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

And the `container` input collect logs from:

```
/var/lib/docker/containers/24f473bc1267.log (100%)
/var/lib/docker/containers/59f473bc1295.log (42%)
```

After this migration we expect that the following files will continue to be ingested by the new `filestream` inputs from their current positions:

```
/var/log/java-exceptions3.log (75%)
/var/log/java-exceptions4.log (0%)
/var/log/java-exceptions5.log (0%)
/var/log/my-application2.json (5%)
/var/log/my-application3.json (0%)
/var/log/my-old-files1.json (0%)
/var/lib/docker/containers/59f473bc1295.log (42%)
```

## Replacing with `filestream`

::::{important}
Don't start Filebeat until you finish steps 1-3 of this guide. Otherwise, you'll see invalid data or data duplication.
The intermediate configuration snippets are for illustration purposes only.
::::

### Step 1: Unique identifiers [step_1]

Every `filestream` input **must** have a unique identifier as `id`. Using meaningful identifiers for each new `filestream` input will also make it easier to troubleshoot if something goes wrong.

::::{important}
Never change the ID of an input, or you will end up with duplicate events.
::::

Let's start the migration process with this simple set of `filestream` inputs without any additional parameters for now:

```yaml
filebeat.inputs:
 - type: filestream
   enabled: true
   id: my-java-collector
   paths:
     - /var/log/java-exceptions*.log

 - type: filestream
   enabled: true
   id: my-container-input
   paths:
     - /var/lib/docker/containers/*.log

 - type: filestream
   enabled: true
   id: my-application-input
   paths:
     - /var/log/my-application*.json

 - type: filestream
   enabled: true
   id: my-old-files
   paths:
     - /var/log/my-old-files*.log
```

### Step 2: The "take over" mode [step_2]

In order to indicate that the new `filestream` inputs are supposed to take over the files from the previously defined `log` or `container` inputs and continue where they left off, we need to activate the "take over" mode by adding `take_over.enabled: true` to each new `filestream`.

::::{important}
For the "take over" mode to work the `paths` list in each new `filestream` inputs must match the `paths` list in each old `log` or `container` inputs accordingly.
::::

After enabling the "take over" mode the configuration should look like this:

```yaml
filebeat.inputs:
 - type: filestream
   enabled: true
   id: my-java-collector
   take_over:
     enabled: true
   paths:
     - /var/log/java-exceptions*.log

 - type: filestream
   enabled: true
   id: my-container-input
   take_over:
     enabled: true
   paths:
     - /var/lib/docker/containers/*.log

 - type: filestream
   enabled: true
   id: my-application-input
   take_over:
     enabled: true
   paths:
     - /var/log/my-application*.json

 - type: filestream
   enabled: true
   id: my-old-files
   take_over:
     enabled: true
   paths:
     - /var/log/my-old-files*.log
```

### Step 3: Migrate additional parameters if necessary [step_3]

Some configuration options are renamed or moved in `filestream`, they are:

| **`log` input** | **`filestream` input** |
| --- | --- |
| harvester_buffer_size | [buffer_size](/reference/filebeat/filebeat-input-filestream.md#_buffer_size) |
| max_bytes | [message_max_bytes](/reference/filebeat/filebeat-input-filestream.md#_message_max_bytes) |
| recursive_glob.enabled | [prospector.scanner.recursive_glob](/reference/filebeat/filebeat-input-filestream.md#filestream-recursive-glob) |
| scan_frequency | [prospector.scanner.check_interval](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-scan-frequency) |
| symlinks | [prospector.scanner.symlinks](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-prospector-scanner-symlinks) |
| exclude_files | [prospector.scanner.exclude_files](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-exclude-files) |
| json | [parsers.n.ndjson](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-ndjson) |
| multiline | [parsers.n.multiline](/reference/filebeat/filebeat-input-filestream.md#_multiline_3) |
| close_inactive | [close.on_state_change.inactive](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-close-inactive) |
| close_removed | [close.on_state_change.removed](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-close-removed) |
| close_inactive | [close.on_state_change.inactive](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-close-inactive) |
| close_eof | [close.reader.on_eof](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-close-eof) |
| close_timeout | [close.reader.after_interval](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-close-timeout) |
| tail_files | [ignore_inactive.since_last_start](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-ignore-inactive) |
| backoff | [backoff.init](/reference/filebeat/filebeat-input-filestream.md#_backoff_init) |
| backoff_max | [backoff.max](/reference/filebeat/filebeat-input-filestream.md#_backoff_max) |

::::{important}
The most significant change is the [parser configuration](/reference/filebeat/filebeat-input-filestream.md#_parsers). Now the ordering is configurable, so `filestream` expects a list of parsers instead.

Furthermore, the `json` parser was renamed to [`ndjson`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-ndjson) to better reflect its functionality and the [`container`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-parsers-container) parser was introduced to replace the additional logic applied by the [`container`](/reference/filebeat/filebeat-input-container.md) input.
::::

The example configuration shown earlier needs to be adjusted according to the changes described in the table above:

```yaml
filebeat.inputs:
 - type: filestream
   enabled: true
   id: my-java-collector
   take_over:
     enabled: true
   paths:
     - /var/log/java-exceptions*.log
   parsers:
     - multiline:
         pattern: '^\['
         negate: true
         match: after
   close.on_state_change.removed: true
   close.on_state_change.renamed: true

 - type: filestream
   enabled: true
   id: my-container-input
   prospector.scanner.symlinks: true # container logs often use symlinks, they should be enabled
   parsers:
     - container: ~ # the container parser replaces everything the container input did before
   take_over:
     enabled: true
   paths:
     - /var/lib/docker/containers/*.log

 - type: filestream
   enabled: true
   id: my-application-input
   take_over:
     enabled: true
   paths:
     - /var/log/my-application*.json
   prospector.scanner.check_interval: 1m
   parsers:
     - ndjson:
         keys_under_root: true

 - type: filestream
   enabled: true
   id: my-old-files
   take_over:
     enabled: true
   paths:
     - /var/log/my-old-files*.log
   ignore_inactive: since_last_start
```

Now you finally have your configuration fully migrated to using `filestream` inputs instead of `log` and `container` inputs.

### Step 4: Validating the migration [step_4]

::::{important}
Double-check that:

* steps 1-3 are correctly performed on your configuration file
* the `log` and `container` inputs you migrated are removed from the configuration
::::

Start Filebeat with the new migrated configuration.

All the events produced by a `filestream` input with `take_over.enabled: true` contain the `take_over` tag. You can filter on this tag in Kibana Discover and see all the events which came from filestreams in the "take over" mode.

Once you start receiving events with this tag and validate that all new `filestream` inputs behave correctly, you can remove `take_over.enabled: true` and restart Filebeat again.

Congratulations, you've completed the migration process.
