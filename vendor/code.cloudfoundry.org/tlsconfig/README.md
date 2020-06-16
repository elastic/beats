# tlsconfig

> build tls configurations

## Reporting issues and requesting features

Please report all issues and feature requests in [cloudfoundry/diego-release](https://github.com/cloudfoundry/diego-release/issues).

## about

There are requirements and guidelines for the TLS configurations
we'd like to use for our internal services. This library stays up to date with
those internal requirements so that services just need to link against this.

This repository also includes a sub-package called `certtest` which can be used
to build valid PKIs for test.

## usage

**Note**: This repository should be imported as `code.cloudfoundry.org/tlsconfig`

See [GoDoc][godoc].

[godoc]: https://godoc.org/code.cloudfoundry.org/tlsconfig

## getting help

Please file an issue!
