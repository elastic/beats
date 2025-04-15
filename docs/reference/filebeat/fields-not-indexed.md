---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/fields-not-indexed.html
---

# Fields are not indexed or usable in Kibana visualizations [fields-not-indexed]

If you have recently performed an operation that loads or parses custom, structured logs, you might need to refresh the index to make the fields available in {{kib}}. To refresh the index, use the [refresh API](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-indices-refresh). For example:

```sh
curl -XPOST 'http://localhost:9200/filebeat-2016.08.09/_refresh'
```

