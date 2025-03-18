---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/_step_4.html
---

# Step 4 [_step_4]

The events produced by `filestream` input with `take_over.enabled:
true` contain a `take_over` tag. You can filter on this tag in Kibana
and see the events which came from a filestream in the "take_over"
mode.

Once you start receiving events with this tag, you can remove
`take_over.enabled: true` and restart the fileinput again.

