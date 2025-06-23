---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/newline-character-required-eof.html
---

# Filebeat isn't shipping the last line of a file [newline-character-required-eof]

Filebeat uses a newline character to detect the end of an event. If lines are added incrementally to a file that’s being harvested, a newline character is required after the last line, or Filebeat will not read the last line of the file.

