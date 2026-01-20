:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/redis/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This module periodically fetches metrics from [Redis](http://redis.io/) servers.

The defaut metricsets are `info` and `keyspace`.


## Module-specific configuration notes [_module_specific_configuration_notes_18]

The Redis module has these additional config options:

**`hosts`**
:   URLs that are used to connect to Redis. URL format: redis://[:password@]host[:port][/db-number][?option=value] redis://HOST[:PORT][?password=PASSWORD[&db=DATABASE]]

**`password`**
:   The password to authenticate, by default it’s empty.

**`idle_timeout`**
:   The duration to remain idle before closing connections. If the value is zero, then idle connections are not closed. The default value is 2 times the module period to allow a connection to be reused across fetches. The `idle_timeout` should be set to less than the server’s connection timeout.

**`network`**
:   The network type to be used for the Redis connection. The default value is `tcp`.

**`maxconn`**
:   The maximum number of concurrent connections to Redis. The default value is 10.


## Compatibility [_compatibility_45]

The redis metricsets `info`, `key` and `keyspace` are compatible with all distributions of Redis (OSS and enterprise). They were tested with Redis 3.2.12, 4.0.11, 5.0.5, 6.2.6, 7.4.7, and 8.2.3, and are expected to work with all versions >= 3.0.

Note: Some fields are only available in specific Redis versions:
- Fields like `used_memory_dataset` were added in Redis 4.0
- Client-side caching tracking stats (`tracking.*`) were added in Redis 6.0
- VM memory fields (`vm.*`), script/function counts, and `used_memory_scripts` were added in Redis 7.0
- The `used_memory_lua` field was deprecated in Redis 7.0 in favor of `vm.eval`
