## 9.4.3 [beats-9.4.3-deprecations]


**Beats**

::::{dropdown} Deprecate ssl.restart_on_cert_change in favor of ssl.certificate_reload.
Certificates, keys, and CA certificates are now automatically reloaded on each TLS handshake without requiring a process restart.

For more information, check [#50444](https://github.com/elastic/beats/pull/50444).
::::

