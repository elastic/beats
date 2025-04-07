---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/_if_something_went_wrong.html
---

# If something went wrong [_if_something_went_wrong]

If for whatever reason you’d like to revert the configuration after running the migrated configuration and return to old `log` inputs the files that were taken by `filestream` inputs, you need to do the following:

1. Stop Filebeat as soon as possible
2. Save its debug-level logs for further investigation
3. Find your [`registry.path/filebeat` directory](/reference/filebeat/configuration-general-options.md#configuration-global-options)
4. Find the created backup files, they have the `<timestamp>.bak` suffix. If you have multiple backups for the same file, choose the one with the more recent timestamp.
5. Replace the files with their backups, e.g. `log.json` should be replaced by `log.json-1674152412247684000.bak`
6. Run Filebeat with the old configuration (no `filestream` inputs with `take_over: true`).

::::{note}
Reverting to backups might cause some events to repeat, depends on the amount of time the new configuration was running.
::::


