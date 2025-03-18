---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/enable-auditbeat-debugging.html
---

# Debug [enable-auditbeat-debugging]

By default, Auditbeat sends all its output to syslog. When you run Auditbeat in the foreground, you can use the `-e` command line flag to redirect the output to standard error instead. For example:

```sh
auditbeat -e
```

The default configuration file is auditbeat.yml (the location of the file varies by platform). You can use a different configuration file by specifying the `-c` flag. For example:

```sh
auditbeat -e -c myauditbeatconfig.yml
```

You can increase the verbosity of debug messages by enabling one or more debug selectors. For example, to view publisher-related messages, start Auditbeat with the `publisher` selector:

```sh
auditbeat -e -d "publisher"
```

If you want all the debugging output (fair warning, it’s quite a lot), you can use `*`, like this:

```sh
auditbeat -e -d "*"
```

