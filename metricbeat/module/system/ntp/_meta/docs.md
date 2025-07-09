The System `ntp` metricset provides Network Time Protocol (NTP) metrics.

This metricset is available on:

* FreeBSD
* Linux
* macOS
* OpenBSD
* Windows

## Configuration [_configuration_5]

**`ntp.host`**
: The remote NTP server address of the form "host", "host:port", "host%zone:port", "[host]:port" or "[host%zone]:port". The host may contain an IPv4, IPv6 or domain name address. When specifying both a port and an IPv6 address, one of the bracket formats must be used. If no port is included, NTP default port 123 is used.

**`ntp.timeout`**
: Timeout determines how long the client waits for a response from the remote server before failing with a timeout error. Defaults to 5 seconds.

**`ntp.version`**
: Version of the NTP protocol to use. Must be one of "3" or "4". Defaults to "4".