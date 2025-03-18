---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/load-ingest-pipelines.html
---

# Load ingest pipelines [load-ingest-pipelines]

The ingest pipelines used to parse log lines are set up automatically the first time you run Filebeat, assuming the {{es}} output is enabled. If you’re sending events to {{ls}} you need to load the ingest pipelines manually. To do this, run the `setup` command with the `--pipelines` option specified.  You also need to enable the modules and filesets, this can be accomplished several ways.

First you can use the `--modules` option to enable the module, and the `-M` option to enable the fileset.  For example, the following command loads the access pipeline from the nginx module.

**deb and rpm:**

```sh
filebeat setup --pipelines --modules nginx -M "nginx.access.enabled=true"
```

**mac:**

```sh
./filebeat setup --pipelines --modules nginx -M "nginx.access.enabled=true"
```

**linux:**

```sh
./filebeat setup --pipelines --modules nginx -M "nginx.access.enabled=true"
```

**win:**

```sh
PS > .\filebeat.exe setup --pipelines --modules nginx -M "nginx.access.enabled=true"
```

The second option is to use the `--modules` option to enable the module, and the `--force-enable-module-filesets` option to enable all the filesets in the module.  For example, the following command loads the access pipeline from the nginx module.

**deb and rpm:**

```sh
filebeat setup --pipelines --modules nginx --force-enable-module-filesets
```

**mac:**

```sh
./filebeat setup --pipelines --modules nginx --force-enable-module-filesets
```

**linux:**

```sh
./filebeat setup --pipelines --modules nginx --force-enable-module-filesets
```

**win:**

```sh
PS > .\filebeat.exe setup --pipelines --modules nginx --force-enable-module-filesets
```

The third option is to use the `--enable-all-filesets` option to enable all the modules and all the filesets so all of the ingest pipelines are loaded.  For example, the following command loads all the ingest pipelines.

**deb and rpm:**

```sh
filebeat setup --pipelines --enable-all-filesets
```

**mac:**

```sh
./filebeat setup --pipelines --enable-all-filesets
```

**linux:**

```sh
./filebeat setup --pipelines --enable-all-filesets
```

**win:**

```sh
PS > .\filebeat.exe setup --pipelines --enable-all-filesets
```

::::{tip}
If you’re loading ingest pipelines manually because you want to send events to {{ls}}, also see [Working with Filebeat modules](logstash://reference/working-with-filebeat-modules.md).
::::


