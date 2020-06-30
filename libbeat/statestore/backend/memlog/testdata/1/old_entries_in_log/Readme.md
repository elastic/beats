Due to restart the log file was not truncated after the last checkpoint was
written. Update with ID 0 adds the removed key0 again to the store. All entries
in log.json should be ignored.
