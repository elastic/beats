---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/master/metricbeat-metricset-windows-wmi.html
---

# Windows wmi metricset [metricbeat-metricset-windows-wmi]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The `wmi` metricset of the Windows module reads metrics via Windows Management Instrumentation  [(WMI)](https://learn.microsoft.com/en-us/windows/win32/wmisdk/about-wmi), a core management technology in the Windows Operating system.

By leveraging WMI Query Language (WQL), this metricset allows you to extract detailed system information and metrics to monitor the health and performance of Windows Systems.

This metricset leverages the [Microsoft WMI](https://github.com/microsoft/wmi), library a convenient wrapper around the [GO-OLE](https://github.com/go-ole) library which allows to invoke the WMI Api.

