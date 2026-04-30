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
