---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/shutdown.html
---

# Stop Metricbeat [shutdown]

An orderly shutdown of Metricbeat ensures that it has a chance to clean up and close outstanding resources. You can help ensure an orderly shutdown by stopping Metricbeat properly.

If you’re running Metricbeat as a service, you can stop it via the service management functionality provided by your installation.

If you’re running Metricbeat directly in the console, you can stop it by entering **Ctrl-C**. Alternatively, send SIGTERM to the Metricbeat process on a POSIX system.

