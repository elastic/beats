## 9.5.7 [beats-release-notes-9.5.7]



### Features and enhancements [beats-9.5.7-features-enhancements]


**All**

* Build the input status report of beat receivers in time linear in the number of inputs. [#53723](https://github.com/elastic/beats/pull/53723) 

  Each status report of a beat receiver built the map of its input statuses with a linear key lookup per input, which is quadratic in the number of inputs. It is now built in one pass, and an input reporting the status and message it already had no longer schedules a report.
  

**Metricbeat**

* Add tier_stats metricset to the autoops_es module to collect the number and the duration of the requests of the search and indexing tiers of an Elasticsearch Serverless project. [#54117](https://github.com/elastic/beats/pull/54117) 


### Fixes [beats-9.5.7-fixes]


**All**

* Update Brotli to 1.2.7 to fix a decoder panic on malformed input. [#54234](https://github.com/elastic/beats/pull/54234) 

  Addresses GHSA-c9gr-gmg4-2vqv, the Go decoder panic related to CVE-2026-111497.
* Reduce retries and repeated error logs from Beats receivers while the output is unreachable. [#54271](https://github.com/elastic/beats/pull/54271) 
* Fix &#34;ApiVersionsRequest V3&#34; errors on Kafka broker connections by updating the sarama fork. [#54546](https://github.com/elastic/beats/pull/54546) 

  Updates the sarama fork to include the upstream fix that corrects how the client version is reported, which caused &#34;ApiVersionsRequest V3&#34; errors when connecting to Kafka brokers.

**Auditbeat**

* Fix auditd module failing or dropping events when `immutable: true` is set. [#53707](https://github.com/elastic/beats/pull/53707) 

  With `immutable: true`, the auditd module could fail at startup with
  `Failure setting audit config as immutable` and stop collecting events, or
  drop audit events that arrived while it waited for the kernel&#39;s reply. The
  request is now sent on the control socket.
  
* Stop auditbeat add_session_metadata stalling audit event reading after falling behind. [#53853](https://github.com/elastic/beats/pull/53853) 

  The kernel_tracing backend of add_session_metadata waited up to 1.2 seconds
  for each process missing from its cache, regardless of how old the event
  was. Once auditbeat fell behind, every event in the backlog was for a process
  that had already exited and waited the full time, so auditbeat never caught
  up, stopped reading the audit socket, and the kernel dropped audit messages.
  The wait is now measured from the event&#39;s timestamp, so events that have
  already waited longer than the limit are not delayed further.
  
* Fix file_integrity kprobes backend failing on kernels that list .cold symbols first. [#54106](https://github.com/elastic/beats/pull/54106) 

  The kprobes backend of the file_integrity module stopped at the first /proc/kallsyms entry whose name started with the probed symbol, so on kernels that list a cold-path fragment such as fsnotify.cold ahead of fsnotify it reported &#34;symbol fsnotify is optimised&#34; and failed to start. Symbol lookup now scans the whole table, prefers an exact match, and ignores .cold fragments, which are not compiler clones of the function.
  
* Fix Auditbeat file_integrity realtime events on macOS when recursive is false. [#54279](https://github.com/elastic/beats/pull/54279) [#54257](https://github.com/elastic/beats/issues/54257)

  Corrected FSEvents file info handling so successful stat calls are not returned as errors. This restores
  realtime change event processing for non-recursive macOS file integrity watches.
  

**Filebeat**

* Fix awss3 input rejecting non_aws_bucket_name configs that set only default_region. [#54143](https://github.com/elastic/beats/pull/54143) 
* Fix processors being applied twice to Elastic Agent filestream inputs configured without streams. [#54416](https://github.com/elastic/beats/pull/54416) 

  When Elastic Agent sent a filestream input whose configuration had no
  streams, the input-level processors were added to the generated input
  configuration twice, so each event ran through them twice. Processors that
  are not idempotent, such as add_tags or script, produced wrong results.
  
* Update mito to v1.27.1, fixing redirects with a body and lost errors in the CEL lazy stream decoders. [#54430](https://github.com/elastic/beats/pull/54430) 
* Write Azure Event Hub v2 checkpoints once per message, after all its records complete. [#54503](https://github.com/elastic/beats/pull/54503) 

  The Azure Event Hub input (processor v2) now writes a checkpoint only after
  every record decoded from a message has been acknowledged by the output or
  dropped by a processor, and at most once per message.
  
  Previously, when processors dropped records with drop_event, the input wrote
  the same checkpoint once per dropped record (for example, 45 writes for a
  message with 45 records), which multiplied Azure Storage operations and made
  consumption wait on storage latency. It could also write a message&#39;s
  checkpoint after the output had acknowledged only some of its records, so a
  restart at that moment skipped the remaining records.
  
* Fix repeated re-reads in azure-blob-storage and gcs inputs with mixed-case file names. [#54512](https://github.com/elastic/beats/pull/54512) 

  A mismatch in case sensitivity between checkpoint selection and filtering caused the azure-blob-storage and gcs inputs to re-read files on every poll when file names mixed upper- and lower-case letters, increasing ingest lag and output load. Affected inputs read those files once more after upgrading, then stop re-reading them.
* Fix Filebeat receivers that share a registry failing to start together on Windows. [#54530](https://github.com/elastic/beats/pull/54530) 

  When several Filebeat receivers in the same process shared a registry path and started at the same time with a new registry, one of them could fail on Windows with &#34;Failed to migrate registry file&#34; and stop without starting its inputs.
* Use the Elasticsearch state store for the aws-cloudwatch input on agentless. [#54573](https://github.com/elastic/beats/pull/54573) 

  The aws-cloudwatch input did not request the Elasticsearch-backed state store with its input type, so on agentless its lastSync checkpoint was kept in the pod-local registry and lost on every pod replacement, causing the whole log group to be reread. The input now requests the store with its type and input ID. Behavior is unchanged when the Elasticsearch state store is not enabled for the input.

**Heartbeat**

* Stop leaking a thread per run when a browser monitor&#39;s synthetics command fails to start. [#52405](https://github.com/elastic/beats/pull/52405) 

**Libbeat**

* Encode the data_stream field set by the add_data_stream processor with lowercase keys in beat receivers. [#54224](https://github.com/elastic/beats/pull/54224) 

**Metricbeat**

* Fix process CPU ticks and percentages being under-reported about 40 times on Apple Silicon macOS. [#54502](https://github.com/elastic/beats/pull/54502) 

  The darwin process collector treated the CPU times reported by proc_pidinfo
  as nanoseconds, but they are in Mach absolute-time units. On Intel the two
  are equal, but on Apple Silicon one unit is 125/3 nanoseconds, so
  system.process.cpu.*.ticks, system.process.cpu.total.value,
  system.process.cpu.total.pct, system.process.cpu.total.norm.pct and
  process.cpu.pct were about 41.67 times too low.
  

**Osquerybeat**

* Stop scheduled query profiling from stalling osquery scheduled results. [#53775](https://github.com/elastic/beats/pull/53775) [#53697](https://github.com/elastic/beats/issues/53697)

  Scheduled query profiles are now collected outside osqueryd&#39;s logger callback, so the osquery_schedule query no longer deadlocks on osqueryd&#39;s schedule mutex. Profiles are published best effort; on the OTel runtime&#39;s slab queue a full queue drops the profile, while on the memory queue a profile publish can wait for a slot, as scheduled result publishes already do. Scheduled query profile documents are now actually published, so ingest volume rises where profiling is enabled. Scheduled profiles carry the new fields schedule_id and schedule_execution_count. Profiles are published after their result, and their @timestamp is the collection time. New counters under osquerybeat.scheduled_query_profiles report collected, failed and dropped profiles.
* Restart osqueryd with backoff when it exits unexpectedly after startup. [#53938](https://github.com/elastic/beats/pull/53938) [#53645](https://github.com/elastic/beats/issues/53645)
* Fix the osquerybeat receiver hanging when it is shut down right after starting. [#53938](https://github.com/elastic/beats/pull/53938) [#53865](https://github.com/elastic/beats/issues/53865)
* Keep osqueryd running when the output is slow by queueing scheduled osquery results. [#54251](https://github.com/elastic/beats/pull/54251) [#53645](https://github.com/elastic/beats/issues/53645)

  Osquerybeat now publishes native scheduled query results from its own queue, so a slow output no longer makes osqueryd exit with code 78. The scheduled queue holds up to osquery.elastic_options.publishing.scheduled.max_events rows (default 3200) and, when full, drops the oldest items; publishing.scheduled.max_age optionally drops items that waited too long. Queued results survive osqueryd restarts and are dropped when osquerybeat stops. Drops are counted under osquerybeat.publishing.scheduled. A policy with invalid publishing settings is rejected before any of it is applied. See x-pack/osquerybeat/docs/publishing.md.
* Reject osquery policies that cannot be applied instead of stopping osquerybeat. [#54249](https://github.com/elastic/beats/pull/54249) 

  An osquery policy that fails validation, for example one with an invalid processor, an invalid ECS mapping or an invalid query schedule, is now rejected before any of it is applied: on the process runtime the osquery unit is marked failed, the previous policy keeps running, and the next valid policy clears the failure. Before, a configuration that failed to parse was ignored and the unit stayed healthy, and a policy osquerybeat could not apply stopped osquerybeat. On the OTel runtime the policy is the receiver&#39;s static configuration and such a configuration still stops the receiver.

**Winlogbeat**

* Fix PowerShell module pipeline failure on unexpected or localized ParameterBinding lines. [#54115](https://github.com/elastic/beats/pull/54115) 

  The powershell and powershell_operational ingest pipelines threw
  &#34;Index 4 out of bounds for length 4&#34; when a ParameterBinding line did not
  match the English-only name/value labels, marking the document with
  event.kind pipeline_error. Unexpected lines are now kept as the raw value,
  and localized labels (e.g. German Name=/Wert=) are parsed.
  
* Fix PowerShell module pipeline failure on event 4103 with multi-line ContextInfo values. [#54260](https://github.com/elastic/beats/pull/54260) 

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
  

