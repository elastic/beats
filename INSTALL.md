## Getting started guide

Best way to get Packetbeat up and running is to follow [this
guide](http://packetbeat.com/getstarted). It installs Packetbeat using the
binaries that we provide for the latest release.

## Installing from source

Packetbeat is written in the Go programming language. Thus, you need to have a
[Go compiler](http://golang.org/) installed before compiling Packetbeat. The
``go get`` setup below also require a set of version control system (Git,
Mercurial and Bazaar) in order to download the source dependencies.

### Compiling Packetbeat

To build Packetbeat successfully, first you need to install all the
dependencies:

    $ go get

The *go build* command is used to compile the package. It only builds the
package, without installing it.

    $ go build

## Run unit tests

Best is to use the Makefile target:

    $ make test

## Install

For installing, you can use our make target:

    $ make install

To install the (optional) GeoIP library, you can use your package manager or,
if you python installed, `pip`:

    $ pip install python-geoip
    $ pip install python-geoip-geolite2

For more information on the GeoIP library, see
[maxmind.com](https://www.maxmind.com/).
