---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/shutdown.html
---

# Stop Heartbeat [shutdown]

An orderly shutdown of Heartbeat ensures that it has a chance to clean up and close outstanding resources. You can help ensure an orderly shutdown by stopping Heartbeat properly.

If you’re running Heartbeat as a service, you can stop it via the service management functionality provided by your installation.

If you’re running Heartbeat directly in the console, you can stop it by entering **Ctrl-C**. Alternatively, send SIGTERM to the Heartbeat process on a POSIX system.

