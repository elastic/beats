---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/shutdown.html
applies_to:
  stack: ga
---

# Stop Filebeat [shutdown]

An orderly shutdown of Filebeat ensures that it has a chance to clean up and close outstanding resources. You can help ensure an orderly shutdown by stopping Filebeat properly.

If you’re running Filebeat as a service, you can stop it via the service management functionality provided by your installation.

If you’re running Filebeat directly in the console, you can stop it by entering **Ctrl-C**. Alternatively, send SIGTERM to the Filebeat process on a POSIX system.

