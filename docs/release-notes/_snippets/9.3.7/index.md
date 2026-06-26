## 9.3.7 [beats-release-notes-9.3.7]



### Features and enhancements [beats-9.3.7-features-enhancements]


**Metricbeat**

* Add MongoDB collstats collection through the $collStats aggregation stage. [#48638](https://github.com/elastic/beats/pull/48638) [#45925](https://github.com/elastic/beats/issues/45925)


### Fixes [beats-9.3.7-fixes]


**All**

* Fix panic when logging.metrics.period is 0. [#51462](https://github.com/elastic/beats/pull/51462) 

**Filebeat**

* Fix WebSocket reconnect loop ignoring context cancellation with infinite retries. [#51434](https://github.com/elastic/beats/pull/51434) [#51520](https://github.com/elastic/beats/pull/51520) [#51512](https://github.com/elastic/beats/pull/51512) [#51580](https://github.com/elastic/beats/pull/51580) [#51581](https://github.com/elastic/beats/pull/51581) [#51582](https://github.com/elastic/beats/pull/51582) [#51583](https://github.com/elastic/beats/pull/51583) [#49267](https://github.com/elastic/beats/issues/49267) [#50904](https://github.com/elastic/beats/issues/50904) [#51046](https://github.com/elastic/beats/issues/51046) [#51123](https://github.com/elastic/beats/issues/51123)
* Fix WebSocket input hanging on shutdown when server stalls and keep_alive is disabled. [#51434](https://github.com/elastic/beats/pull/51434) [#51520](https://github.com/elastic/beats/pull/51520) [#51512](https://github.com/elastic/beats/pull/51512) [#51580](https://github.com/elastic/beats/pull/51580) [#51581](https://github.com/elastic/beats/pull/51581) [#51582](https://github.com/elastic/beats/pull/51582) [#51583](https://github.com/elastic/beats/pull/51583) [#49267](https://github.com/elastic/beats/issues/49267) [#50904](https://github.com/elastic/beats/issues/50904) [#51046](https://github.com/elastic/beats/issues/51046) [#51123](https://github.com/elastic/beats/issues/51123)
* Fix handling of user-agent header when using OAuth2.0 authentication. [#51434](https://github.com/elastic/beats/pull/51434) [#51520](https://github.com/elastic/beats/pull/51520) [#51512](https://github.com/elastic/beats/pull/51512) [#51580](https://github.com/elastic/beats/pull/51580) [#51581](https://github.com/elastic/beats/pull/51581) [#51582](https://github.com/elastic/beats/pull/51582) [#51583](https://github.com/elastic/beats/pull/51583) [#49267](https://github.com/elastic/beats/issues/49267) [#50904](https://github.com/elastic/beats/issues/50904) [#51046](https://github.com/elastic/beats/issues/51046) [#51123](https://github.com/elastic/beats/issues/51123)
* Fix data race on filestream registry cursor metadata. [#51434](https://github.com/elastic/beats/pull/51434) [#51520](https://github.com/elastic/beats/pull/51520) [#51512](https://github.com/elastic/beats/pull/51512) [#51580](https://github.com/elastic/beats/pull/51580) [#51581](https://github.com/elastic/beats/pull/51581) [#51582](https://github.com/elastic/beats/pull/51582) [#51583](https://github.com/elastic/beats/pull/51583) [#49267](https://github.com/elastic/beats/issues/49267) [#50904](https://github.com/elastic/beats/issues/50904) [#51046](https://github.com/elastic/beats/issues/51046) [#51123](https://github.com/elastic/beats/issues/51123)
* Strip sensitive headers on cross-origin redirects in httpjson and CEL inputs. [#51434](https://github.com/elastic/beats/pull/51434) [#51520](https://github.com/elastic/beats/pull/51520) [#51512](https://github.com/elastic/beats/pull/51512) [#51580](https://github.com/elastic/beats/pull/51580) [#51581](https://github.com/elastic/beats/pull/51581) [#51582](https://github.com/elastic/beats/pull/51582) [#51583](https://github.com/elastic/beats/pull/51583) [#49267](https://github.com/elastic/beats/issues/49267) [#50904](https://github.com/elastic/beats/issues/50904) [#51046](https://github.com/elastic/beats/issues/51046) [#51123](https://github.com/elastic/beats/issues/51123)

  When redirect.forward_headers is true, the Authorization, Proxy-Authorization,
  and Cookie headers are now automatically removed on redirects that cross to a
  different host or downgrade from HTTPS to HTTP. A new redirect.sensitive_headers
  configuration option controls which headers are stripped; set it to [] to restore
  the previous behaviour of forwarding all headers unconditionally.
  
* Validate request tracer path regardless of enabled state to prevent unintended file deletion. [#51434](https://github.com/elastic/beats/pull/51434) [#51520](https://github.com/elastic/beats/pull/51520) [#51512](https://github.com/elastic/beats/pull/51512) [#51580](https://github.com/elastic/beats/pull/51580) [#51581](https://github.com/elastic/beats/pull/51581) [#51582](https://github.com/elastic/beats/pull/51582) [#51583](https://github.com/elastic/beats/pull/51583) [#49267](https://github.com/elastic/beats/issues/49267) [#50904](https://github.com/elastic/beats/issues/50904) [#51046](https://github.com/elastic/beats/issues/51046) [#51123](https://github.com/elastic/beats/issues/51123)

**Heartbeat**

* Restore `tls.*` certificate metadata for HTTP monitors using `max_redirects &gt; 0`. [#51434](https://github.com/elastic/beats/pull/51434) [#51520](https://github.com/elastic/beats/pull/51520) [#51512](https://github.com/elastic/beats/pull/51512) [#51580](https://github.com/elastic/beats/pull/51580) [#51581](https://github.com/elastic/beats/pull/51581) [#51582](https://github.com/elastic/beats/pull/51582) [#51583](https://github.com/elastic/beats/pull/51583) [#48335](https://github.com/elastic/beats/issues/48335)

  HTTP monitors that follow redirects (or use a proxy) stopped exporting `tls.*` fields after the 8.0 migration to elastic-agent-libs, because the shared HTTP transport wrapped the TLS connection in a way that prevented Go&#39;s net/http from populating `Response.TLS`. This restores TLS certificate metadata (including certificate expiry) for redirecting HTTPS endpoints.
  

**Libbeat**

* Fix panic in the Kafka output during shutdown. [#51434](https://github.com/elastic/beats/pull/51434) [#51520](https://github.com/elastic/beats/pull/51520) [#51512](https://github.com/elastic/beats/pull/51512) [#51580](https://github.com/elastic/beats/pull/51580) [#51581](https://github.com/elastic/beats/pull/51581) [#51582](https://github.com/elastic/beats/pull/51582) [#51583](https://github.com/elastic/beats/pull/51583) [#49267](https://github.com/elastic/beats/issues/49267) [#50904](https://github.com/elastic/beats/issues/50904) [#51046](https://github.com/elastic/beats/issues/51046) [#51123](https://github.com/elastic/beats/issues/51123)

