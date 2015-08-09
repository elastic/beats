# Beats Packer

Tools, scripts and docker images for cross-compiling and packaging the Elastic
[Beats](https://www.elastic.co/products/beats).

## Prepare

You need Go, python and docker installed. Prepare the rest with:

     make deps

Python is only used for the [j2cli](https://github.com/kolypto/j2cli) tool,
which we might replace with a Go implementation.

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

        build/packetbeat-1.0.0-nightly.150809100849-darwin.tgz
        build/packetbeat-1.0.0-nightly.150809100849-darwin.tgz.sha1
        build/packetbeat-1.0.0-nightly.150809100849-i686.rpm
        build/packetbeat-1.0.0-nightly.150809100849-i686.rpm.sha1
        build/packetbeat-1.0.0-nightly.150809100849-x86_64.rpm
        build/packetbeat-1.0.0-nightly.150809100849-x86_64.rpm.sha1
        build/packetbeat-1.0.0-nightly.150809100857-windows.zip
        build/packetbeat-1.0.0-nightly.150809100857-windows.zip.sha1
        build/packetbeat_1.0.0-nightly.150809100849_amd64.deb
        build/packetbeat_1.0.0-nightly.150809100849_amd64.deb.sha1
        build/packetbeat_1.0.0-nightly.150809100849_i386.deb
        build/packetbeat_1.0.0-nightly.150809100849_i386.deb.sha1
