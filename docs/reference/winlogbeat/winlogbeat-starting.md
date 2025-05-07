---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/winlogbeat-starting.html
---

# Start Winlogbeat [winlogbeat-starting]

Before starting Winlogbeat:

* Follow the steps in [Quick start: installation and configuration](/reference/winlogbeat/winlogbeat-installation-configuration.md) to install, configure, and set up the Winlogbeat environment.
* Make sure {{kib}} and {{es}} are running.
* Make sure the user specified in `winlogbeat.yml` is [authorized to publish events](/reference/winlogbeat/privileges-to-publish-events.md).

To start Winlogbeat, run:

```shell
PS C:\Program Files\Winlogbeat> Start-Service winlogbeat
```

Winlogbeat should now be running. If you used the logging configuration described here, you can view the log file at `C:\ProgramData\winlogbeat\Logs\winlogbeat`.

You can view the status of the service and control it from the Services management console in Windows. To launch the management console, run this command:

```shell
PS C:\Program Files\Winlogbeat> services.msc
```

