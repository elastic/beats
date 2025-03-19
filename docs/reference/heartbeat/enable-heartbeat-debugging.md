---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/enable-heartbeat-debugging.html
---

# Debug [enable-heartbeat-debugging]

By default, Heartbeat sends all its output to syslog. When you run Heartbeat in the foreground, you can use the `-e` command line flag to redirect the output to standard error instead. For example:

```sh
heartbeat -e
```

The default configuration file is heartbeat.yml (the location of the file varies by platform). You can use a different configuration file by specifying the `-c` flag. For example:

```sh
heartbeat -e -c myheartbeatconfig.yml
```

You can increase the verbosity of debug messages by enabling one or more debug selectors. For example, to view publisher-related messages, start Heartbeat with the `publisher` selector:

```sh
heartbeat -e -d "publisher"
```

If you want all the debugging output (fair warning, it’s quite a lot), you can use `*`, like this:

```sh
heartbeat -e -d "*"
```

