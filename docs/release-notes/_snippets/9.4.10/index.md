## 9.4.10 [beats-release-notes-9.4.10]



### Features and enhancements [beats-9.4.10-features-enhancements]


**All**

* Build the input status report of beat receivers in time linear in the number of inputs. [#54792](https://github.com/elastic/beats/pull/54792) 

  Each status report of a beat receiver built the map of its input statuses with a linear key lookup per input, which is quadratic in the number of inputs. It is now built in one pass, and an input reporting the status and message it already had no longer schedules a report.
  

**Metricbeat**

* Add tier_stats metricset to the autoops_es module to collect the number and the duration of the requests of the search and indexing tiers of an Elasticsearch Serverless project. [#54792](https://github.com/elastic/beats/pull/54792) 


### Fixes [beats-9.4.10-fixes]


**All**

* Update Brotli to 1.2.7 to fix a decoder panic on malformed input. [#54792](https://github.com/elastic/beats/pull/54792) 

  Addresses GHSA-c9gr-gmg4-2vqv, the Go decoder panic related to CVE-2026-111497.
* Fix quadratic cost of building the receiver config for units with many streams. [#54792](https://github.com/elastic/beats/pull/54792) 
* Fix &#34;ApiVersionsRequest V3&#34; errors on Kafka broker connections by updating the sarama fork. [#54546](https://github.com/elastic/beats/pull/54546) 

  Updates the sarama fork to include the upstream fix that corrects how the client version is reported, which caused &#34;ApiVersionsRequest V3&#34; errors when connecting to Kafka brokers.

**Auditbeat**

* Fix auditd module failing or dropping events when `immutable: true` is set. [#54792](https://github.com/elastic/beats/pull/54792) 

  With `immutable: true`, the auditd module could fail at startup with
  `Failure setting audit config as immutable` and stop collecting events, or
  drop audit events that arrived while it waited for the kernel&#39;s reply. The
  request is now sent on the control socket.
  
* Stop auditbeat add_session_metadata stalling audit event reading after falling behind. [#54792](https://github.com/elastic/beats/pull/54792) 

  The kernel_tracing backend of add_session_metadata waited up to 1.2 seconds
  for each process missing from its cache, regardless of how old the event
  was. Once auditbeat fell behind, every event in the backlog was for a process
  that had already exited and waited the full time, so auditbeat never caught
  up, stopped reading the audit socket, and the kernel dropped audit messages.
  The wait is now measured from the event&#39;s timestamp, so events that have
  already waited longer than the limit are not delayed further.
  
* Fix file_integrity kprobes backend failing on kernels that list .cold symbols first. [#54792](https://github.com/elastic/beats/pull/54792) 

  The kprobes backend of the file_integrity module stopped at the first /proc/kallsyms entry whose name started with the probed symbol, so on kernels that list a cold-path fragment such as fsnotify.cold ahead of fsnotify it reported &#34;symbol fsnotify is optimised&#34; and failed to start. Symbol lookup now scans the whole table, prefers an exact match, and ignores .cold fragments, which are not compiler clones of the function.
  
* Fix Auditbeat file_integrity realtime events on macOS when recursive is false. [#54279](https://github.com/elastic/beats/pull/54279) [#54257](https://github.com/elastic/beats/issues/54257)

  Corrected FSEvents file info handling so successful stat calls are not returned as errors. This restores
  realtime change event processing for non-recursive macOS file integrity watches.
  

**Filebeat**

* Fix processors being applied twice to Elastic Agent filestream inputs configured without streams. [#54792](https://github.com/elastic/beats/pull/54792) 

  When Elastic Agent sent a filestream input whose configuration had no
  streams, the input-level processors were added to the generated input
  configuration twice, so each event ran through them twice. Processors that
  are not idempotent, such as add_tags or script, produced wrong results.
  
* Fix repeated re-reads in azure-blob-storage and gcs inputs with mixed-case file names. [#54792](https://github.com/elastic/beats/pull/54792) 

  A mismatch in case sensitivity between checkpoint selection and filtering caused the azure-blob-storage and gcs inputs to re-read files on every poll when file names mixed upper- and lower-case letters, increasing ingest lag and output load. Affected inputs read those files once more after upgrading, then stop re-reading them.
* Fix Filebeat receivers that share a registry failing to start together on Windows. [#54792](https://github.com/elastic/beats/pull/54792) 

  When several Filebeat receivers in the same process shared a registry path and started at the same time with a new registry, one of them could fail on Windows with &#34;Failed to migrate registry file&#34; and stop without starting its inputs.

**Heartbeat**

* Stop leaking a thread per run when a browser monitor&#39;s synthetics command fails to start. [#54792](https://github.com/elastic/beats/pull/54792) 

**Libbeat**

* Encode the data_stream field set by the add_data_stream processor with lowercase keys in beat receivers. [#54792](https://github.com/elastic/beats/pull/54792) 
* Accept client certificates signed by CAs added to `certificate_authorities` while a TLS server is running. [#54792](https://github.com/elastic/beats/pull/54792) 

  With `ssl.certificate_reload` enabled (the default when `certificate_authorities`
  is set), a running TLS server reloaded `certificate_authorities` from disk but
  still rejected client certificates signed by a CA added after startup, because
  the Go TLS stack verified them against the CA pool captured when the server
  was configured. Removing a CA already took effect. The server now uses the
  current CA pool for each handshake, both to verify client certificates and to
  advertise acceptable CAs to clients.
  

**Metricbeat**

* Fix process CPU ticks and percentages being under-reported about 40 times on Apple Silicon macOS. [#54792](https://github.com/elastic/beats/pull/54792) 

  The darwin process collector treated the CPU times reported by proc_pidinfo
  as nanoseconds, but they are in Mach absolute-time units. On Intel the two
  are equal, but on Apple Silicon one unit is 125/3 nanoseconds, so
  system.process.cpu.*.ticks, system.process.cpu.total.value,
  system.process.cpu.total.pct, system.process.cpu.total.norm.pct and
  process.cpu.pct were about 41.67 times too low.
  

**Winlogbeat**

* Fix PowerShell module pipeline failure on unexpected or localized ParameterBinding lines. [#54792](https://github.com/elastic/beats/pull/54792) 

  The powershell and powershell_operational ingest pipelines threw
  &#34;Index 4 out of bounds for length 4&#34; when a ParameterBinding line did not
  match the English-only name/value labels, marking the document with
  event.kind pipeline_error. Unexpected lines are now kept as the raw value,
  and localized labels (e.g. German Name=/Wert=) are parsed.
  
* Fix PowerShell module pipeline failure on event 4103 with multi-line ContextInfo values. [#54792](https://github.com/elastic/beats/pull/54792) 

  The powershell_operational ingest pipeline failed with &#34;field
  [winlog.event_data.ContextInfo] does not contain value_split [[:=]]&#34; when
  Host Application held a multi-line script, for example one passed to
  powershell.exe -Command. The ContextInfo block is now parsed by a script
  that uses its fixed 16-key layout, so the full script is kept in
  process.command_line and script lines that look like keys are not taken as
  keys. Localized keys, including ones with non-ASCII letters or apostrophes,
  are still parsed as separate fields, and dots are removed from event_data
  keys (for example the Spanish &#34;Id. de host&#34; becomes &#34;Iddehost&#34;) so they
  are not read as object paths.
  

