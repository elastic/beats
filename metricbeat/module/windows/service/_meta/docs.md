The `service` metricset of the Windows module reads the status of Windows services.


## Dashboard [_dashboard_47]

The service metricset comes with a predefined dashboard. For example:

![metricbeat windows service](images/metricbeat-windows-service.png)


## Configuration [_configuration_20]

```yaml
- module: windows
  metricsets: ["service"]
  period: 60s
```


## Filtering [_filtering_2]

Processors can be used to filter the events based on the service states or their names. The example below configures the metricset to drop all events except for the events for the firewall service. See [Processors](/reference/metricbeat/filtering-enhancing-data.md) for more information about using processors.

```yaml
- module: windows
  metricsets: ["service"]
  period: 60s
  processors:
    - drop_event.when.not.equals:
        windows.service.display_name: Windows Firewall
```
