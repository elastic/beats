---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/enable-packetbeat-debugging.html
---

# Debug [enable-packetbeat-debugging]

By default, Packetbeat sends all its output to syslog. When you run Packetbeat in the foreground, you can use the `-e` command line flag to redirect the output to standard error instead. For example:

```sh
packetbeat -e
```

The default configuration file is packetbeat.yml (the location of the file varies by platform). You can use a different configuration file by specifying the `-c` flag. For example:

```sh
packetbeat -e -c mypacketbeatconfig.yml
```

You can increase the verbosity of debug messages by enabling one or more debug selectors. For example, to view publisher-related messages, start Packetbeat with the `publisher` selector:

```sh
packetbeat -e -d "publisher"
```

If you want all the debugging output (fair warning, it’s quite a lot), you can use `*`, like this:

```sh
packetbeat -e -d "*"
```

