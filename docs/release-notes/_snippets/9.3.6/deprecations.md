## 9.3.6 [beats-9.3.6-deprecations]


**Beats**

::::{dropdown} Deprecate ssl.restart_on_cert_change in favor of ssl.certificate_reload.
Certificates, keys, and CA certificates are now automatically reloaded on each TLS handshake without requiring a process restart.

For more information, check [#50444](https://github.com/elastic/beats/pull/50444)[#34074](https://github.com/elastic/beats/issues/34074).

% **Impact**<br>_Add a description of the impact_

% **Action**<br>_Add a description of the what action to take_
::::

