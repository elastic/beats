The `tag` metricset reads the configured PLC tags once per period and creates one event per tag.

Each tag is read in its own request, so a tag that can't be read doesn't hide the values of the other tags. Such a tag produces an event with its response code and an error message, and without a value. When no tag can be read, the connection is closed and opened again on the next period.

The value is stored in one of the typed fields `value.double`, `value.long`, `value.boolean`, `value.keyword` or `value.date`, depending on its IEC 61131-3 data type, so that each field keeps a single type across tags. Lists, structures, durations and times of day are not indexed; their data type is still reported.
