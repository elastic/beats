---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/shutdown.html
---

# Stop Packetbeat [shutdown]

An orderly shutdown of Packetbeat ensures that it has a chance to clean up and close outstanding resources. You can help ensure an orderly shutdown by stopping Packetbeat properly.

If you’re running Packetbeat as a service, you can stop it via the service management functionality provided by your installation.

If you’re running Packetbeat directly in the console, you can stop it by entering **Ctrl-C**. Alternatively, send SIGTERM to the Packetbeat process on a POSIX system.

