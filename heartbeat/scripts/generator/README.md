## Readme

Code generator for Heartbeat monitors.

In order to add a new monitor type to Heartbeat, run `make create-monitor
MONITOR=<name>` from the Heartbeat directory.

```
$ cd $GOPATH/src/github.com/elastic/beats/heartbeat
$ make create-monitor MONITOR=<name>
$ make update
$ make # build heartbeat
```

`make update` is required to update the import list, such that the new monitor
is compiled into Heartbeat.

The new monitor will be added to the `monitors/active/<name>` sub directory.

Monitor structure:
- `config.go`: The monitor configuration options and configuration validation.
- `check.go`: The monitor validation support.
- `job.go`: Implements the ping function for connecting and validating an
  endpoint. This file generates the monitor specific fields to an event.
- `<name>.go`: The monitor entrypoint registering the factory for setting up
  monitoring jobs.
- `_meta/fields.yml`: Document the monitors event field.
- `_meta/config.yml`: Minimal sample configuration file 
- `_meta/config.reference.yml`: Reference configuration file. All availalbe
  settings are documented in the reference file.

Code comments tagged with `IMPLEMENT_ME` in the go and meta files give details
on changes required to implement a new monitor type.
