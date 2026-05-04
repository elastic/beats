## 9.4.0 [beats-release-notes-9.4.0]



### Features and enhancements [beats-9.4.0-features-enhancements]


**All**

* Export all beat receiver metrics to OTel telemetry. [#49300](https://github.com/elastic/beats/pull/49300) 
* Add add_agent_metadata processor to inject agent metadata efficiently. [#49667](https://github.com/elastic/beats/pull/49667) 
* Update OTel Collector components to v0.149.0/v1.55.0. [#50057](https://github.com/elastic/beats/pull/50057) 

**Elastic agent**

* Logstash exporter now reports accurate error status to EDOT. [#49169](https://github.com/elastic/beats/pull/49169) 

**Filebeat**

* Add lexicographical polling mode to AWS-S3 input. [#48310](https://github.com/elastic/beats/pull/48310) [#47926](https://github.com/elastic/beats/issues/47926)
* Optimize opening files in filestream for better performance. [#48506](https://github.com/elastic/beats/pull/48506) 
* Instruments the CEL Input with OpenTelemetry tracing. [#48440](https://github.com/elastic/beats/pull/48440) 
* Add ES state store routing for lexicographical mode. [#48944](https://github.com/elastic/beats/pull/48944) 
* Add experimental bbolt-based registry backend for Filebeat. [#48879](https://github.com/elastic/beats/pull/48879) 

  Add a new bbolt (BoltDB) storage backend for the Filebeat registry,
  configurable via `filebeat.registry.backend: bbolt`. The bbolt backend
  provides persistent on-disk storage with support for database compaction
  and TTL-based entry cleanup.
  This feature is experimental and may change or be removed in future releases.
  
* Add MFA enrichment support to the azure-ad entity analytics provider. [#49843](https://github.com/elastic/beats/pull/49843) 

  A new optional enrich_with setting has been added to the azure-ad entity analytics
  input. When set to [&#34;mfa&#34;], the provider fetches MFA registration details for all
  users from the Microsoft Graph API endpoint
  /reports/authenticationMethods/userRegistrationDetails and merges them into each
  user document under the azure_ad.mfa field. This requires the AuditLog.Read.All
  application permission in Azure.
  
* Add a querylog fileset to the Filebeat elasticsearch module for NDJSON query logs. [#49914](https://github.com/elastic/beats/pull/49914) [#43622](https://github.com/elastic/beats/issues/43622)

  Adds a dedicated `querylog` fileset that tails Elasticsearch query log files (`*_querylog.json`),
  one JSON object per line, using the filestream input with an NDJSON parser (`expand_keys`).
  
  The fileset collects structured query events for DSL, ES|QL, EQL, and SQL (timings, shard
  information, optional ES|QL profile phases, remote-cluster metadata where present, and related
  ECS fields such as user and trace). Module field definitions cover `elasticsearch.querylog.*` plus
  `elasticsearch.task.id`, `elasticsearch.parent.task.id`, and `elasticsearch.parent.node.id`.
  
  Default log paths follow the usual Elasticsearch layout on Linux, macOS, and Windows; users can
  override them with `var.paths`. The fileset is disabled by default in the module reference config.
  Documentation describes how to enable it and set paths.
  
* Optimize filestream to allocate less memory when applying include/exclude line filters. [#49013](https://github.com/elastic/beats/pull/49013) 
* Add capacity to collect empty Active Directory groups to entity analytics input. [#49093](https://github.com/elastic/beats/pull/49093) 
* Add input redirection support through a new Redirector mechanism. [#49613](https://github.com/elastic/beats/pull/49613) 
* Allow HTTP JSON input redirection to the CEL input. [#49614](https://github.com/elastic/beats/pull/49614) 
* Update mito CEL library to v1.25.1 and cel-go runtime to v0.27.0. [#49683](https://github.com/elastic/beats/pull/49683) 
* Add `perms` enrichment option to the Okta entity analytics provider to collect permissions for custom roles. [#49805](https://github.com/elastic/beats/pull/49805) [#49779](https://github.com/elastic/beats/issues/49779)

  A new `perms` value is available in the `enrich_with` configuration option of
  the Okta entity analytics input. When enabled, role permissions are fetched from
  the Okta IAM API for each custom role assigned to a user and stored under
  `roles[].permissions` in the published event. Because permissions depend on role
  data, including `perms` implicitly enables role enrichment. Only custom roles
  (type `CUSTOM`) are queried; standard built-in roles are skipped. This option
  requires the `okta.roles.read` OAuth2 scope and introduces additional API calls
  per user, so it should be enabled with care on large tenants.
  
* Add devices enrichment option to Okta entity analytics provider. [#49813](https://github.com/elastic/beats/pull/49813) [#49780](https://github.com/elastic/beats/issues/49780)

  Adds devices as a new optional value for the enrich_with configuration
  option in the Okta entity analytics provider. When enabled, each user is enriched
  with the list of devices enrolled for that user via the List User Devices Okta API
  endpoint. The enrichment is opt-in and excluded from the default configuration to
  avoid the extra per-user API call that would increase Okta rate limit consumption.
  
* Add supervises enrichment option to the Okta entity analytics provider. [#49825](https://github.com/elastic/beats/pull/49825) [#49781](https://github.com/elastic/beats/issues/49781)

  Add a new &#34;supervises&#34; value to the enrich_with option of the Okta entity analytics
  provider. When enabled, each user document is enriched with a supervises field containing
  the list of users they manage. Each entry includes the managed user&#39;s id, profile.email,
  and profile.login (the Okta username). The list is derived by querying the Okta API for
  users whose profile.managerId matches the manager&#39;s user ID. The option is disabled by
  default because it requires one additional API call per user, which may exceed Okta rate
  limits in large deployments.
  

**Filebeat, metricbeat**

* Add NewFactoryWithSettings for Beat receivers to provide default home and path directories. [#49327](https://github.com/elastic/beats/pull/49327) [#11734](https://github.com/elastic/elastic-agent/issues/11734)

**Heartbeat**

* Heartbeat-custom-policy-reload. [#49326](https://github.com/elastic/beats/pull/49326) [#47511](https://github.com/elastic/beats/issues/47511)

  Add custom policy hashing and live-update functionality to integrations

**Metricbeat**

* Add cursor-based incremental data fetching to the SQL module query metricset. [#48722](https://github.com/elastic/beats/pull/48722) 

  Add a cursor feature to the SQL query metricset that enables incremental data
  fetching by tracking the last fetched row value. Supports integer, timestamp,
  date, float, and decimal cursor types with ascending and descending scan
  directions. State is persisted via libbeat statestore (memlog backend).
  
* Add switchport_statuses config option to filter Meraki switchports by status. [#47993](https://github.com/elastic/beats/pull/47993) 
* Add the subexpiry field to the Redis INFO Keyspace (Redis ≥ 7.4). [#47971](https://github.com/elastic/beats/pull/47971) [#26555](https://github.com/elastic/enhancements/issues/26555)
* Add Redis 6.0/7.0 info fields and deprecate used_memory_lua. [#48246](https://github.com/elastic/beats/pull/48246) 

  Updates the Redis info metricset to support fields from Redis 6.0 and 7.0:
  
  Redis 7.0 memory fields:
  - Added memory.vm.eval, vm.functions, vm.total (VM memory not counted in used_memory)
  - Added memory.used.scripts, used.scripts_eval, used.functions (script memory overhead)
  - Added memory.total_system (total system memory)
  - Added server.number_of_cached_scripts, number_of_functions, number_of_libraries
  
  Redis 6.0 fields:
  - Added stats.tracking.total_keys, total_items, total_prefixes (client-side caching stats)
  
  Other changes:
  - Marked used_memory_lua as deprecated (replaced by vm.eval in Redis 7.0)
  - Made used_memory_lua and used_memory_dataset optional for older Redis compatibility
  
* Add `state` field to IPSec tunnel metrics in panw module. [#48403](https://github.com/elastic/beats/pull/48403) 
* Map Docker network metrics to different types for better usability. [#47792](https://github.com/elastic/beats/pull/47792) 
* Add observer hostname field to panw module. [#48825](https://github.com/elastic/beats/pull/48825) 
* Bump azure-sdk-for-go armmonitor from v0.8.0 to v0.11.0. [#49866](https://github.com/elastic/beats/pull/49866) 

**Osquerybeat**

* Jumplists table for osquery extension. [#47759](https://github.com/elastic/beats/pull/47759) 
* Add gentables code generator for creating typed Go packages from YAML specs. [#48533](https://github.com/elastic/beats/pull/48533) 
* Adding automatic jumplists to elastic_jumplists table. [#48032](https://github.com/elastic/beats/pull/48032) 
* Add elastic_host_processes and elastic_host_users tables and host_processes, host_users views. [#48794](https://github.com/elastic/beats/pull/48794) 
* Add optional query profiling for scheduled and live osquery runs. [#49514](https://github.com/elastic/beats/pull/49514) 
* Updates osquerybeat filters to avoid reliance on type assertion. [#48540](https://github.com/elastic/beats/pull/48540) 
* Allow for passing of osqueryd client to extension tables. [#48544](https://github.com/elastic/beats/pull/48544) 
* Gentables improvements and elastic_browser_history spec with registry integration. [#48733](https://github.com/elastic/beats/pull/48733) 
* Migrate elastic_file_analysis table to osquery table spec and dedicated package. [#48774](https://github.com/elastic/beats/pull/48774) 

  The elastic_file_analysis osquery table (macOS) is now defined by the unified YAML spec
  in ext/osquery-extension/specs/elastic_file_analysis.yaml. Implementation lives in
  pkg/fileanalysis with generated table code in pkg/tables/generated/elastic_file_analysis.
  Registration is via the generated darwin registry; no behavior change.
  
* Add elastic_host_groups table, host_groups view, and default view hooks in generator. [#48775](https://github.com/elastic/beats/pull/48775) 

  - Renamed host_groups table to elastic_host_groups; implementation moved to pkg/hostgroups.
  - host_groups is now a deprecated view (SELECT * FROM elastic_host_groups) for backward compatibility.
  - View generator now produces create/delete hooks by default and RegisterDefaultViewHook(hm); custom
    hooks can still be registered via RegisterHooksFunc (e.g. default &#43; extra, like amcache).
  
* Add amcache table and view specs and generated tables/view for osquery extension. [#48802](https://github.com/elastic/beats/pull/48802) 

  Amcache tables and the elastic_amcache_applications view are now defined by YAML specs
  (ext/osquery-extension/specs/elastic_amcache_*.yaml) with group amcache. Generated table
  and view code live in pkg/tables/generated/amcache and pkg/views/generated/amcache.
  Glue (generate funcs, view hooks, cleanup) is in pkg/amcache/register_generated.go.
  Unused methods and non-generated docs were removed.
  
* Add native scheduled query metadata and schedule_id correlation fields. [#49040](https://github.com/elastic/beats/pull/49040) 

  Native scheduled query outputs now include deterministic scheduling metadata:
  `schedule_execution_count` and `planned_schedule_time`.
  
  Scheduled query outputs also use `schedule_id` for correlation, while live
  action query outputs continue to use `action_id`.
  
  Scheduled response documents include `pack_id` for pack queries when present
  in the configuration.
  
  Native scheduler option defaults remain available through `osquery.options`
  (`schedule_splay_percent` and `schedule_max_drift`).
  
* Add elastic_jumplists spec integration and harden osquerybeat generation workflow. [#49058](https://github.com/elastic/beats/pull/49058) 

  - Added elastic_jumplists table spec and shared types, generated table/docs wiring, and
    migrated jumplists glue to generated RegisterGenerateFunc/GetGenerateFunc flow.
  - Extended generated table registry/getter signatures to pass ResilientClient and wired it
    through osquery-extension table registration.
  - Added generated osquery-extension README support in gentables to keep tables/views docs in sync.
  - Added osquerybeat mage Generate target and made Check depend on it for generation consistency.
  - Updated jumplists generator to use local cached source files by default, with on-demand refresh
    via `-refresh-sources` (and `JUMPLISTS_REFRESH_SOURCES=true mage generate`), plus enforced
    gofmt/goimports formatting of generated jumplists files.
  
* Add support for per-platform custom osqueryd artifact install in osquerybeat. [#49306](https://github.com/elastic/beats/pull/49306) [#48955](https://github.com/elastic/beats/issues/48955)


### Fixes [beats-9.4.0-fixes]


**Agentbeat**

* Update transient dependency github.com/go-jose/go-jose/v4 to v4.1.4. [#49975](https://github.com/elastic/beats/pull/49975) 

**All**

* Update to Go 1.25.9. [#50049](https://github.com/elastic/beats/pull/50049) 
* Bump aws-sdk-go-v2/service/cloudwatchlogs to v1.65.0 to fix GHSA-xmrv-pmrh-hhx2. [#50215](https://github.com/elastic/beats/pull/50215) 

**Filebeat**

* Support abuse.ch auth key usage in the Threat Intel module. [#45212](https://github.com/elastic/beats/pull/45212) [#45206](https://github.com/elastic/beats/issues/45206)
* Fix max_body_bytes setting not working without HMAC and add missing documentation configuration options in HTTP Endpoint input. [#48550](https://github.com/elastic/beats/pull/48550) [#48512](https://github.com/elastic/beats/issues/48512)

  Previously, the max_body_bytes setting was only applied during HMAC validation, meaning it had no effect on requests that didn&#39;t use HMAC authentication.
  This fix ensures that body size limiting is applied to all incoming requests regardless of authentication method.
  Additionally, restored missing documentation for the max_body_bytes setting in the HTTP Endpoint input.
  
* Fix http_endpoint input shared server lifecycle causing joiner deadlock and creator killing unrelated inputs. [#49415](https://github.com/elastic/beats/pull/49415) 

  Decouple the shared HTTP server lifetime from any single input. Previously,
  the server context was derived from the creator input, so cancelling a joiner
  blocked forever (deadlock) and cancelling the creator shut down all inputs on
  the same port. The server now lives until the last input deregisters.
  
* Fix typo in CEL input OTel tracing logging. [#49692](https://github.com/elastic/beats/pull/49692) [#49625](https://github.com/elastic/beats/issues/49625)
* Fix container input not respecting max bytes when parsing CRI partial lines. [#49743](https://github.com/elastic/beats/pull/49743) [#49259](https://github.com/elastic/beats/issues/49259)
* Fix internal processing time metric for azureeventhub input. [#40547](https://github.com/elastic/beats/pull/40547) 
* Fix CSV decoder producing malformed JSON when field values contain double quotes in azure-blob-storage input. [#50097](https://github.com/elastic/beats/pull/50097) 

  The azure-blob-storage input&#39;s decode path only matched the decoder.Decoder
  interface, which builds JSON via string concatenation without escaping field
  values. CSV values containing double quotes (e.g. RFC 2045 MIME type
  parameters) produce malformed JSON, causing downstream ingest pipeline
  failures. Add a decoder.ValueDecoder switch case which uses json.Marshal
  for correct escaping, matching the pattern already used by the GCS input.
  
* Update cel-go to v0.28.0, fixing runtime error location reporting. [#50176](https://github.com/elastic/beats/pull/50176) 
* Re-evaluate url_program on each websocket reconnect using evolved cursor state. [#50383](https://github.com/elastic/beats/pull/50383) 

  The streaming input now re-evaluates url_program before each websocket
  reconnection (both error recovery and OAuth2 token refresh), allowing
  cursor state accumulated during the session to influence the reconnect URL.
  Previously url_program was evaluated once at startup and the result was
  reused for all subsequent connections. The process function also now returns
  the evolved cursor so that callers can propagate it into the shared state.
  
* Reduce allocation pressure in httpjson cursor update and split paths. [#50384](https://github.com/elastic/beats/pull/50384) 

**Libbeat**

* Fix conversion of time duration fields such as event.duration when using Beats receivers. [#50302](https://github.com/elastic/beats/pull/50302) 

**Metricbeat**

* AutoOps ES module update to use UUID v7 without dashes to reduce payloads. [#50078](https://github.com/elastic/beats/pull/50078) 

**Osquerybeat**

* Fix jumplist table to ensure embedded fields are exported. [#49649](https://github.com/elastic/beats/pull/49649) 
* Avoid mutating osquery install config during validation to prevent races. [#49769](https://github.com/elastic/beats/pull/49769) 

**Packetbeat**

* Fix janitor goroutine leaks and decoder cleanup lifecycle on route changes. [#48836](https://github.com/elastic/beats/pull/48836) 

**Winlogbeat**

* Fix no_more_events stop losing final batch of events when io.EOF is returned alongside records. [#49012](https://github.com/elastic/beats/pull/49012) [#47388](https://github.com/elastic/beats/issues/47388)

