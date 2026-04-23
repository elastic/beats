## 9.3.4 [beats-release-notes-9.3.4]



### Features and enhancements [beats-9.3.4-features-enhancements]


**All**

* Update OTel Collector components to v0.149.0/v1.55.0. [#50137](https://github.com/elastic/beats/pull/50137) [#49836](https://github.com/elastic/beats/pull/49836) [#50222](https://github.com/elastic/beats/pull/50222) [#50228](https://github.com/elastic/beats/pull/50228) [#50191](https://github.com/elastic/beats/pull/50191) [#49838](https://github.com/elastic/beats/pull/49838) [#50254](https://github.com/elastic/beats/pull/50254) [#50078](https://github.com/elastic/beats/pull/50078) [#50267](https://github.com/elastic/beats/pull/50267) [#50261](https://github.com/elastic/beats/pull/50261) [#50270](https://github.com/elastic/beats/pull/50270) [#50282](https://github.com/elastic/beats/pull/50282) [#50283](https://github.com/elastic/beats/pull/50283) [#50284](https://github.com/elastic/beats/pull/50284) [#50285](https://github.com/elastic/beats/pull/50285) [#49803](https://github.com/elastic/beats/issues/49803) [#49803](https://github.com/elastic/beats/issues/49803) [#50077](https://github.com/elastic/beats/issues/50077) [#50217](https://github.com/elastic/beats/issues/50217)

**Metricbeat**

* Bump azure-sdk-for-go armmonitor from v0.8.0 to v0.11.0. [#49866](https://github.com/elastic/beats/pull/49866) 


### Fixes [beats-9.3.4-fixes]


**Agentbeat**

* Update transient dependency github.com/go-jose/go-jose/v4 to v4.1.4. [#50137](https://github.com/elastic/beats/pull/50137) [#49836](https://github.com/elastic/beats/pull/49836) [#50222](https://github.com/elastic/beats/pull/50222) [#50228](https://github.com/elastic/beats/pull/50228) [#50191](https://github.com/elastic/beats/pull/50191) [#49838](https://github.com/elastic/beats/pull/49838) [#50254](https://github.com/elastic/beats/pull/50254) [#50078](https://github.com/elastic/beats/pull/50078) [#50267](https://github.com/elastic/beats/pull/50267) [#50261](https://github.com/elastic/beats/pull/50261) [#50270](https://github.com/elastic/beats/pull/50270) [#50282](https://github.com/elastic/beats/pull/50282) [#50283](https://github.com/elastic/beats/pull/50283) [#50284](https://github.com/elastic/beats/pull/50284) [#50285](https://github.com/elastic/beats/pull/50285) [#49803](https://github.com/elastic/beats/issues/49803) [#49803](https://github.com/elastic/beats/issues/49803) [#50077](https://github.com/elastic/beats/issues/50077) [#50217](https://github.com/elastic/beats/issues/50217)

**Filebeat**

* Fix http_endpoint input shared server lifecycle causing joiner deadlock and creator killing unrelated inputs. [#50137](https://github.com/elastic/beats/pull/50137) [#49836](https://github.com/elastic/beats/pull/49836) [#50222](https://github.com/elastic/beats/pull/50222) [#50228](https://github.com/elastic/beats/pull/50228) [#50191](https://github.com/elastic/beats/pull/50191) [#49838](https://github.com/elastic/beats/pull/49838) [#50254](https://github.com/elastic/beats/pull/50254) [#50078](https://github.com/elastic/beats/pull/50078) [#50267](https://github.com/elastic/beats/pull/50267) [#50261](https://github.com/elastic/beats/pull/50261) [#50270](https://github.com/elastic/beats/pull/50270) [#50282](https://github.com/elastic/beats/pull/50282) [#50283](https://github.com/elastic/beats/pull/50283) [#50284](https://github.com/elastic/beats/pull/50284) [#50285](https://github.com/elastic/beats/pull/50285) [#49803](https://github.com/elastic/beats/issues/49803) [#49803](https://github.com/elastic/beats/issues/49803) [#50077](https://github.com/elastic/beats/issues/50077) [#50217](https://github.com/elastic/beats/issues/50217)

  Decouple the shared HTTP server lifetime from any single input. Previously,
  the server context was derived from the creator input, so cancelling a joiner
  blocked forever (deadlock) and cancelling the creator shut down all inputs on
  the same port. The server now lives until the last input deregisters.
  
* Fix container input not respecting max bytes when parsing CRI partial lines. [#50137](https://github.com/elastic/beats/pull/50137) [#49836](https://github.com/elastic/beats/pull/49836) [#50222](https://github.com/elastic/beats/pull/50222) [#50228](https://github.com/elastic/beats/pull/50228) [#50191](https://github.com/elastic/beats/pull/50191) [#49838](https://github.com/elastic/beats/pull/49838) [#50254](https://github.com/elastic/beats/pull/50254) [#50078](https://github.com/elastic/beats/pull/50078) [#50267](https://github.com/elastic/beats/pull/50267) [#50261](https://github.com/elastic/beats/pull/50261) [#50270](https://github.com/elastic/beats/pull/50270) [#50282](https://github.com/elastic/beats/pull/50282) [#50283](https://github.com/elastic/beats/pull/50283) [#50284](https://github.com/elastic/beats/pull/50284) [#50285](https://github.com/elastic/beats/pull/50285) [#49259](https://github.com/elastic/beats/issues/49259)
* Fix CSV decoder producing malformed JSON when field values contain double quotes in azure-blob-storage input. [#50137](https://github.com/elastic/beats/pull/50137) [#49836](https://github.com/elastic/beats/pull/49836) [#50222](https://github.com/elastic/beats/pull/50222) [#50228](https://github.com/elastic/beats/pull/50228) [#50191](https://github.com/elastic/beats/pull/50191) [#49838](https://github.com/elastic/beats/pull/49838) [#50254](https://github.com/elastic/beats/pull/50254) [#50078](https://github.com/elastic/beats/pull/50078) [#50267](https://github.com/elastic/beats/pull/50267) [#50261](https://github.com/elastic/beats/pull/50261) [#50270](https://github.com/elastic/beats/pull/50270) [#50282](https://github.com/elastic/beats/pull/50282) [#50283](https://github.com/elastic/beats/pull/50283) [#50284](https://github.com/elastic/beats/pull/50284) [#50285](https://github.com/elastic/beats/pull/50285) [#50097](https://github.com/elastic/beats/issues/50097)

  The azure-blob-storage input&#39;s decode path only matched the decoder.Decoder
  interface, which builds JSON via string concatenation without escaping field
  values. CSV values containing double quotes (e.g. RFC 2045 MIME type
  parameters) produce malformed JSON, causing downstream ingest pipeline
  failures. Add a decoder.ValueDecoder switch case which uses json.Marshal
  for correct escaping, matching the pattern already used by the GCS input.
  
* Fix conflicting CEL periodic OTel metric field names. [#50135](https://github.com/elastic/beats/50135) [#49180](https://github.com/elastic/beats/issues/49180)

  Rename the CEL periodic run counter from input.cel.periodic.run to
  input.cel.periodic.run.count so the run namespace stays consistent
  alongside input.cel.periodic.run.duration in Elasticsearch mappings.
  Also correct related metric documentation and instrument creation
  error messages.
  
* Update mito to v1.24.2, fixing runtime error location reporting. [#50137](https://github.com/elastic/beats/pull/50137) [#49836](https://github.com/elastic/beats/pull/49836) [#50222](https://github.com/elastic/beats/pull/50222) [#50228](https://github.com/elastic/beats/pull/50228) [#50191](https://github.com/elastic/beats/pull/50191) [#49838](https://github.com/elastic/beats/pull/49838) [#50254](https://github.com/elastic/beats/pull/50254) [#50078](https://github.com/elastic/beats/pull/50078) [#50267](https://github.com/elastic/beats/pull/50267) [#50261](https://github.com/elastic/beats/pull/50261) [#50270](https://github.com/elastic/beats/pull/50270) [#50282](https://github.com/elastic/beats/pull/50282) [#50283](https://github.com/elastic/beats/pull/50283) [#50284](https://github.com/elastic/beats/pull/50284) [#50285](https://github.com/elastic/beats/pull/50285) [#49803](https://github.com/elastic/beats/issues/49803) [#49803](https://github.com/elastic/beats/issues/49803) [#50077](https://github.com/elastic/beats/issues/50077) [#50217](https://github.com/elastic/beats/issues/50217)

