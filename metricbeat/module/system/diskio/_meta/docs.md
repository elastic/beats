The System `diskio` metricset provides disk IO metrics collected from the operating system. One event is created for each disk mounted on the system.

This metricset is available on:

* Linux
* macOS (requires 10.10+)
* Windows
* FreeBSD (amd64)


## Configuration [_configuration_6]

**`diskio.include_devices`**
:   When the `diskio` metricset is enabled, you can use the `diskio.include_devices` option to define a list of device names to pre-filter the devices that are reported. Filters only exact matches. If not set or given `[]` empty array, all disk devices are returned

    The following example config returns metrics for devices matching include_devices:

    ```yaml
    metricbeat.modules:
    - module: system
      metricsets: ["diskio"]
      diskio.include_devices: ["sda", "sda1"]
    ```

