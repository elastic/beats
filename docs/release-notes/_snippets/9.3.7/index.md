## 9.3.7 [beats-release-notes-9.3.7]



### Features and enhancements [beats-9.3.7-features-enhancements]


**Metricbeat**

* Add MongoDB collstats collection through the $collStats aggregation stage. [#48638](https://github.com/elastic/beats/pull/48638) [#45925](https://github.com/elastic/beats/issues/45925)


### Fixes [beats-9.3.7-fixes]


**All**

* Fix panic when logging.metrics.period is 0. [#51462](https://github.com/elastic/beats/pull/51462) 

**Filebeat**

* Fix WebSocket reconnect loop ignoring context cancellation with infinite retries. [#51194](https://github.com/elastic/beats/pull/51194) 
* Fix WebSocket input hanging on shutdown when server stalls and keep_alive is disabled. [#51227](https://github.com/elastic/beats/pull/51227) [#51213](https://github.com/elastic/beats/issues/51213)
* Fix handling of user-agent header when using OAuth2.0 authentication. [#51228](https://github.com/elastic/beats/pull/51228) [#50867](https://github.com/elastic/beats/issues/50867)
* Fix data race on filestream registry cursor metadata. [#51287](https://github.com/elastic/beats/pull/51287) 
* Strip sensitive headers on cross-origin redirects in httpjson and CEL inputs. [#51434](https://github.com/elastic/beats/pull/51434) 

  When redirect.forward_headers is true, the Authorization, Proxy-Authorization,
  and Cookie headers are now automatically removed on redirects that cross to a
  different host or downgrade from HTTPS to HTTP. A new redirect.sensitive_headers
  configuration option controls which headers are stripped; set it to [] to restore
  the previous behaviour of forwarding all headers unconditionally.
  
* Validate request tracer path regardless of enabled state to prevent unintended file deletion. [#51479](https://github.com/elastic/beats/pull/51479) 

**Heartbeat**

* Restore `tls.*` certificate metadata for HTTP monitors using `max_redirects &gt; 0`. [#51339](https://github.com/elastic/beats/pull/51339) [#48335](https://github.com/elastic/beats/issues/48335)

  HTTP monitors that follow redirects (or use a proxy) stopped exporting `tls.*` fields after the 8.0 migration to elastic-agent-libs, because the shared HTTP transport wrapped the TLS connection in a way that prevented Go&#39;s net/http from populating `Response.TLS`. This restores TLS certificate metadata (including certificate expiry) for redirecting HTTPS endpoints.
  

**Libbeat**

* Fix panic in the Kafka output during shutdown. [#51484](https://github.com/elastic/beats/pull/51484) 

