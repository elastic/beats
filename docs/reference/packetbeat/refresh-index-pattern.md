---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/refresh-index-pattern.html
---

# Fields show up as nested JSON in Kibana [refresh-index-pattern]

When Packetbeat exports a field of type dictionary, and the keys are not known in advance, the Discovery page in {{kib}} will display the field as a nested JSON object:

```shell
http.response.headers = {
        "content-length": 12,
        "content-type": "application/json"
}
```

To fix this you need to [reload the index pattern](docs-content://explore-analyze/find-and-organize/data-views.md) in {{kib}} under the Management→Index Patterns, and the index pattern will be updated with a field for each key available in the dictionary:

```shell
http.response.headers.content-length = 12
http.response.headers.content-type = "application/json"
```

