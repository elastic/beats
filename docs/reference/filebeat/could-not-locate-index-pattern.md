---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/could-not-locate-index-pattern.html
---

# Dashboard could not locate the index-pattern [could-not-locate-index-pattern]

Typically Filebeat sets up the index pattern automatically when it loads the index template. However, if for some reason Filebeat loads the index template, but the index pattern does not get created correctly, you’ll see a "could not locate that index-pattern" error. To resolve this problem:

1. Try running the `setup` command again. For example: `./filebeat setup`.
2. If that doesn’t work, go to the Management app in {{kib}}, and under **Index Patterns**, look for the pattern.

    1. If the pattern doesn’t exist, create it manually.

        * Set the **Time filter field name** to `@timestamp`.
        * Set the **Custom index pattern ID** advanced option. For example, if your custom index name is `filebeat-customname`, set the custom index pattern ID to `filebeat-customname-*`.


For more information, see [Creating an index pattern](docs-content://explore-analyze/find-and-organize/data-views.md) in the {{kib}} docs.

