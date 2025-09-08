---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/generate-index-pattern.html
---

# Generating the Beat Index Pattern [generate-index-pattern]

The index-pattern defines the format of each field, and it’s used by Kibana to know how to display the field. If you change the fields exported by the Beat, you need to generate a new index pattern for your Beat. Otherwise, you can just use the index pattern available under the `kibana/*/index-pattern` directory.

The Beat index pattern is generated from the `fields.yml`, which contains all the fields exported by the Beat. For each field, besides the `type`, you can configure the `format` field. The format informs Kibana about how to display a certain field. A good example is `percentage` or `bytes` to display fields as `50%` or `5MB`.

To generate the index pattern from the `fields.yml`, you need to run the following command in the Beat repository:

```shell
make update
```

