---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/enable-filebeat-debugging.html
---

# Debug [enable-filebeat-debugging]

By default, Filebeat sends all its output to syslog. When you run Filebeat in the foreground, you can use the `-e` command line flag to redirect the output to standard error instead. For example:

```sh
filebeat -e
```

The default configuration file is filebeat.yml (the location of the file varies by platform). You can use a different configuration file by specifying the `-c` flag. For example:

```sh
filebeat -e -c myfilebeatconfig.yml
```

You can increase the verbosity of debug messages by enabling one or more debug selectors. For example, to view publisher-related messages, start Filebeat with the `publisher` selector:

```sh
filebeat -e -d "publisher"
```

If you want all the debugging output (fair warning, it’s quite a lot), you can use `*`, like this:

```sh
filebeat -e -d "*"
```

