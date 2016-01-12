[![Build Status](https://travis-ci.org/elastic/beats-packer.svg)](https://travis-ci.org/elastic/beats-packer)

# Beats Packer

Tools, scripts and docker images for cross-compiling and packaging the Elastic
[Beats](https://www.elastic.co/products/beats).

## Prepare

You need Go and docker installed. This project uses several docker files, you
can either build them with:

     make images

Or pull them from the Docker registry with:

     make pull-images

Prepare the rest with:

     make deps

## Cross-compile

The cross compilation part is based on [xgo](https://github.com/karalabe/xgo),
with some [changes](https://github.com/tsg/xgo) that add a bit more
extensibility that we needed for the Beats (e.g. static compiling, custom
docker image).

You can cross-compile one Beat for all platforms with (e.g.):

     make packetbeat

## Packaging

For each OS (named platform here) we execute a `build.sh` script which is
free to do whatever it is required to build the proper packages for that
platform. This can include running docker containers with the right tools
included or with that OS installed for native packaging.

The deb and rpm creation is based on [fpm](https://github.com/jordansissel/fpm)
which is executed from a container.

Besides the platform, there are three other dimensions: architecture,
beat and the release. Each of these is defined by YAML files in their folders.
These dimensions only set static options, the platforms is the only one
scripted.

The runner is currently (ab)using a Makefile, which is nice because it can
parallelize things automatically, but it's hacky so we might replace it in
a future.

Building all Beats for all platforms:

     make clean && make -j2

Which currently produces the following:

        topbeat-1.0.0-nightly.150813173034-darwin.tgz
        topbeat-1.0.0-nightly.150813173034-darwin.tgz.sha1
        topbeat_1.0.0-nightly.150813173034_i386.deb
        topbeat_1.0.0-nightly.150813173034_i386.deb.sha1
        topbeat-1.0.0-nightly.150813173034-windows.zip
        topbeat-1.0.0-nightly.150813173034-windows.zip.sha1
        topbeat-1.0.0-nightly.150813173034-i686.rpm
        topbeat-1.0.0-nightly.150813173034-i686.rpm
        topbeat_1.0.0-nightly.150813173034_amd64.deb
        topbeat_1.0.0-nightly.150813173034_amd64.deb.sha1
        topbeat-1.0.0-nightly.150813173034-x86_64.rpm
        topbeat-1.0.0-nightly.150813173034-x86_64.rpm
        packetbeat-1.0.0-nightly.150813173058-windows.zip
        packetbeat-1.0.0-nightly.150813173058-windows.zip.sha1
        packetbeat-1.0.0-nightly.150813173058-darwin.tgz
        packetbeat-1.0.0-nightly.150813173058-darwin.tgz.sha1
        packetbeat_1.0.0-nightly.150813173058_i386.deb
        packetbeat_1.0.0-nightly.150813173058_i386.deb.sha1
        packetbeat-1.0.0-nightly.150813173058-i686.rpm
        packetbeat-1.0.0-nightly.150813173058-i686.rpm
        packetbeat_1.0.0-nightly.150813173058_amd64.deb
        packetbeat_1.0.0-nightly.150813173058_amd64.deb.sha1
        packetbeat-1.0.0-nightly.150813173058-x86_64.rpm
        packetbeat-1.0.0-nightly.150813173058-x86_64.rpm
