---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/reading-from-evtx.html
---

# Not sure how to read from .evtx files [reading-from-evtx]

Yes, Winlogbeat can ingest archived .evtx files. When you set the `name` parameter as the absolute path to an event log file it will read from that file. Here’s an example. First create a new config file for Winlogbeat.

winlogbeat-evtx.yml

```yaml
winlogbeat.event_logs:
  - name: ${EVTX_FILE} <1>
    no_more_events: stop <2>

winlogbeat.shutdown_timeout: 30s <3>
winlogbeat.registry_file: evtx-registry.yml <4>

output.elasticsearch.hosts: ['http://localhost:9200']
```

1. `name` will be set to the value of the `EVTX_FILE` environment variable.
2. `no_more_events` sets the behavior of Winlogbeat when Windows reports that there are no more events to read. We want Winlogbeat to *stop* rather than *wait* since this is an archived file that will not receive any more events.
3. `shutdown_timeout` controls the maximum amount of time Winlogbeat will wait to finish publishing the events to {{es}} after stopping because it reached the end of the log.
4. A separate registry file is used to avoid overwriting the default registry file. You can delete this file after you’re done ingesting the .evtx data.

Now execute Winlogbeat and wait for it to complete. It will exit when it’s done.

```sh
.\winlogbeat.exe -e -c .\winlogbeat-evtx.yml -E EVTX_FILE=c:\backup\Security-2019.01.evtx
```

