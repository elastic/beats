# My TODO

 - definitive test of OTel file log receiver:
   - [x] does it keep state
   - [x] does it keep state of gzip files
   - [x] does it can actually read gzip files "from the end"?
 - Vector: "You should also have zero ambiguity about what Vector does."
   - Vector and Splunk decompress them for reading. Vector does not explain what it means by that, -> Just read the source and/or just test it.
 - [x] file rotation from plain to gzip. "When a file is rotated and replaced with a gzip compressed alternative, we still have a handle open to the uncompressed version even if it is "deleted" correct? Or we can be configured to do this?"
   - [x] how file rotation works right now?
   - [x] is there any other file identity besides fingerprint that can cover
 plain -> gzip rotation?
 - [x] Verification plan: use https://github.com/elastic/benchbuilder to check the
performance


## Vector

Checking the code, it uses [`MultiGzDecoder`](https://docs.rs/flate2/latest/flate2/bufread/struct.MultiGzDecoder.html#impl-Read-for-MultiGzDecoder%3CR%3E)
which, similar to Go, says to stream the data, but it only considers a success
when all the data is read:

> A gzip streaming decoder that decodes a gzip file that may have multiple members.
> [...]
> [...] MultiGzDecoder decodes all members from the data and only returns Ok(0)
> when the underlying reader does. For a file, this reads to the end of the file.

## Filelog receiver general findings:

 - It does not have a graceful shutdown, therefore if interrupted while reading
a files, it will not be able to read the file from the last position when it
restarts. This is valid for both plain and gzip files.
 - It correctly handles:
   - a gzip file is read to the end (file: current.gz)
   - the collector is stopped
   - a new gzip file is appended to the original file (`cat more.gz >> current.gz`)
   - the collector is started again
   - it read only the new data
 - When handling gzipped files, the offset is from the decompressed data, which
is latter used to "seek" into the compressed file. os.FIle.Seek does not error
if the offset is greater than the file size. That is most likely why the
collector can read gzip files to the end and not reingest data if more data is
appended.
 - Bad error handling, it indefinitely logs an error if configured to read gzip
but the file is not gzip.

```text
2025-04-01T09:51:41.343+0200	error	reader/reader.go:84	failed to create gzip reader	{"kind": "receiver", "name": "filelog", "data_type": "logs", "component": "fileconsumer", "path": "/tmp/beats/current.log", "error": "gzip: invalid header"}
github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer/internal/reader.(*Reader).ReadToEnd
	/home/ainsoph/devel/go/pkg/mod/github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza@v0.119.0/fileconsumer/internal/reader/reader.go:84
github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer.(*Manager).consume.func1
	/home/ainsoph/devel/go/pkg/mod/github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza@v0.119.0/fileconsumer/file.go:169
```

### testing it

 - working directory: `/tmp/beats`
 - create a log file with 4242 lines:
   - `docker run -it --rm mingrammer/flog -t log -o ./4242.log -f json -w -n 4242`
 - gzip the file
   - `gzip -k 4242.log`
 - otel cfg:
```yaml
extensions:
  file_storage:
    directory: /tmp/beats/filelog

receivers:
  filelog:
    include:
      - /tmp/beats/4242.log.gz
    start_at: beginning
    storage: file_storage
    compression: gzip

exporters:
  file:
    path: /tmp/beats/output.ndjson
    append: true
  debug:
    verbosity: detailed

service:
  extensions: [file_storage]
  pipelines:
    logs:
      receivers: [filelog]
      exporters: [file, debug]
```
 - start the agent in otel mode with the above comfig
```shell
./elastic-agent otel --config /home/ainsoph/devel/github.com/elastic/beats/x-pack/filebeat/filebeat-anderson-otel.yml
```
 - let it read the whole file
 - check the output file:
   - `wc -l output.ndjson` -> `43 output.ndjson`. It creates a json for each 100 logs.
  - `jq '.resourceLogs[0].scopeLogs[0].logRecords | length' /tmp/beats/output.ndjson`
```text
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
42
```
 - 4242 lines read
 - now let's try to interrupt the read and start again
 - `rm output.ndjson filelog/receiver_filelog_`
 - start the agent and stop it as soon as you see the debug output
```shell
./elastic-agent otel --config /home/ainsoph/devel/github.com/elastic/beats/x-pack/filebeat/filebeat-anderson-otel.yml
```
 - confirm it did not ingest everything:
 - `wc -l output.ndjson` -> `28 output.ndjson`. `wc -l` counts `\n`, so there is one more line in `output.ndjson`
 - `jq '.resourceLogs[0].scopeLogs[0].logRecords | length' /tmp/beats/output.ndjson`
```text
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
```
 - 29 lines -> 2900 logs read
 - start the agent again and let it read to the end. Check the output
 - `wc -l output.ndjson` -> `71 output.ndjson`, 72 lines
```text
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
42
```
 - it read 7142 lines. The 1st 2900 + 4242 (the whole file again)


### Plain file
 - config file
```yaml
extensions:
  file_storage:
    directory: /tmp/beats/filelog

receivers:
  filelog:
    include:
      - /tmp/beats/4242.log
    start_at: beginning
    storage: file_storage

exporters:
  file:
    path: /tmp/beats/output.ndjson
    append: true
  debug:
    verbosity: detailed

service:
  extensions: [file_storage]
  pipelines:
    logs:
      receivers: [filelog]
      exporters: [file, debug]
```
 - `wc -l output.ndjson` -> `6 output.ndjson`
 - `jq '.resourceLogs[0].scopeLogs[0].logRecords | length' /tmp/beats/output.ndjson`
```text
100
100
100
100
100
100
100
```
 - read 700 line
 - start again and let it read til the end
 - `wc -l output.ndjson` -> `49 output.ndjson`
 - `jq '.resourceLogs[0].scopeLogs[0].logRecords | length' /tmp/beats/output.ndjson`
```text
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
100
42
```
 - read 4942, the 1st 700 + 4242
