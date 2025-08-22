The System `ntp` metricset provides Network Time Protocol (NTP) metrics.

This metricset is available on:

* FreeBSD
* Linux
* macOS
* OpenBSD
* Windows

## Configuration [_configuration_5]

**`ntp.servers`**
:   Remote NTP server addresses of the form "host", "host:port", "host%zone:port", "[host]:port" or "[host%zone]:port". The server may contain an IPv4, IPv6, or domain name address. When specifying both a port and an IPv6 address, one of the bracket formats must be used. If no port is included, NTP default port 123 is used. If multiple servers are specified, metrics are reported separately for each. Defaults to ["pool.ntp.org"].

**`ntp.timeout`**
:   Timeout determines how long the client waits for a response from the remote server before failing with a timeout error. Defaults to 5 seconds.

**`ntp.version`**
:   Version of the NTP protocol to use. Must be one of "3" or "4". Defaults to "4".

**`ntp.validate`**
:   Whether to validate the NTP response if it is suitable for time synchronization purposes. If not, fail with a validation error. Must be one of "true" or "false". Defaults to "true".