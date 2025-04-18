---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/enable-winlogbeat-debugging.html
---

# Debug [enable-winlogbeat-debugging]

By default, Winlogbeat sends all its output to syslog. When you run Winlogbeat in the foreground, you can use the `-e` command line flag to redirect the output to standard error instead. For example:

```sh
winlogbeat -e
```

The default configuration file is winlogbeat.yml (the location of the file varies by platform). You can use a different configuration file by specifying the `-c` flag. For example:

```sh
winlogbeat -e -c mywinlogbeatconfig.yml
```

You can increase the verbosity of debug messages by enabling one or more debug selectors. For example, to view publisher-related messages, start Winlogbeat with the `publisher` selector:

```sh
winlogbeat -e -d "publisher"
```

If you want all the debugging output (fair warning, it’s quite a lot), you can use `*`, like this:

```sh
winlogbeat -e -d "*"
```

