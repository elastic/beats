---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/_debugging_on_kibana.html
---

# Debugging on Kibana [_debugging_on_kibana]

Events produced by `filestream` with `take_over.enabled: true` contains `take_over` tag. You can filter on this tag in Kibana and see the events which came from a filestream in the "take over" mode.

