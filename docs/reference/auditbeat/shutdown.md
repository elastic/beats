---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/shutdown.html
---

# Stop Auditbeat [shutdown]

An orderly shutdown of Auditbeat ensures that it has a chance to clean up and close outstanding resources. You can help ensure an orderly shutdown by stopping Auditbeat properly.

If you’re running Auditbeat as a service, you can stop it via the service management functionality provided by your installation.

If you’re running Auditbeat directly in the console, you can stop it by entering **Ctrl-C**. Alternatively, send SIGTERM to the Auditbeat process on a POSIX system.

