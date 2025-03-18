---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/shutdown.html
---

# Stop Winlogbeat [shutdown]

An orderly shutdown of Winlogbeat ensures that it has a chance to clean up and close outstanding resources. You can help ensure an orderly shutdown by stopping Winlogbeat properly.

If you’re running Winlogbeat as a service, you can stop it via the service management functionality provided by your installation.

If you’re running Winlogbeat directly in the console, you can stop it by entering **Ctrl-C**. Alternatively, send SIGTERM to the Winlogbeat process on a POSIX system.

