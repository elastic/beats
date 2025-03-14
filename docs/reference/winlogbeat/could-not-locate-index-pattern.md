---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/could-not-locate-index-pattern.html
---

# Dashboard could not locate the index-pattern [could-not-locate-index-pattern]

Typically Winlogbeat sets up the index pattern automatically when it loads the index template. However, if for some reason Winlogbeat loads the index template, but the index pattern does not get created correctly, you’ll see a "could not locate that index-pattern" error. To resolve this problem:

1. Try running the `setup` command again. For example: `./winlogbeat setup`.
2. If that doesn’t work, go to the Management app in {{kib}}, and under **Index Patterns**, look for the pattern.

    1. If the pattern doesn’t exist, create it manually.

        * Set the **Time filter field name** to `@timestamp`.
        * Set the **Custom index pattern ID** advanced option. For example, if your custom index name is `winlogbeat-customname`, set the custom index pattern ID to `winlogbeat-customname-*`.


For more information, see [Creating an index pattern](docs-content://explore-analyze/find-and-organize/data-views.md) in the {{kib}} docs.

