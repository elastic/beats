---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/enable-metricbeat-debugging.html
---

# Debug [enable-metricbeat-debugging]

By default, Metricbeat sends all its output to syslog. When you run Metricbeat in the foreground, you can use the `-e` command line flag to redirect the output to standard error instead. For example:

```sh
metricbeat -e
```

The default configuration file is metricbeat.yml (the location of the file varies by platform). You can use a different configuration file by specifying the `-c` flag. For example:

```sh
metricbeat -e -c mymetricbeatconfig.yml
```

You can increase the verbosity of debug messages by enabling one or more debug selectors. For example, to view publisher-related messages, start Metricbeat with the `publisher` selector:

```sh
metricbeat -e -d "publisher"
```

If you want all the debugging output (fair warning, it’s quite a lot), you can use `*`, like this:

```sh
metricbeat -e -d "*"
```

