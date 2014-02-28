## Installing from sources

Packetbeat is written in the Go programming language. Thus, you need to have a Go compiler
installed before compiling Packetbeat.

### Compiling Packetbeat

To install GeoIP:

    $ pip install python-geoip
    $ pip install python-geoip-geolite2

It requires the package to be checked out in the local source tree. To get the sources from github use:

    $ go get github.com/packetbeat/packetbeat

To build Packetbeat successfully, first you need to install all the dependencies:

    $ go get github.com/mattbaird/elastigo/api
    $ go get github.com/mattbaird/elastigo/core

The *go build* command is used to compile the package. It only builds the package, without its installation.

    $ go build

