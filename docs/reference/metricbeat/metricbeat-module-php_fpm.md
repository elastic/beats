---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-php_fpm.html
---

# PHP_FPM module [metricbeat-module-php_fpm]

This module periodically fetches metrics from [PHP-FPM](https://php-fpm.org) servers.

The default metricset is `pool`.


## Module-specific configuration notes [_module_specific_configuration_notes_16]

You need to enable the PHP-FPM status page by properly configuring `pm.status_path`.

Here is a sample nginx configuration to forward requests to the PHP-FPM status page (assuming `pm.status_path` is configured with default value `/status`):

```
nginx
location ~ /status {
     allow 127.0.0.1;
     deny all;
     include fastcgi_params;
     fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
     fastcgi_pass 127.0.0.1:9000;
}
```


## Compatibility [_compatibility_42]

The PHP_FPM metricsets were tested with PHP 7.1.1 and are expected to work with all versions >= 5.


## Example configuration [_example_configuration_53]

The PHP_FPM module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: php_fpm
  metricsets:
  - pool
  #- process
  enabled: true
  period: 10s
  status_path: "/status"
  hosts: ["localhost:8080"]
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_62]

The following metricsets are available:

* [pool](/reference/metricbeat/metricbeat-metricset-php_fpm-pool.md)
* [process](/reference/metricbeat/metricbeat-metricset-php_fpm-process.md)



