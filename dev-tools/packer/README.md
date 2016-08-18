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

     make clean && make

## Naming conventions

We use a set of package name conventions across all the Elastic stack:

* The general form is `name-version-os-arch.ext`. Note that this means we
  use dashes even for Deb files.
* The archs are called `x86` and `x64` except for deb/rpm where we keep the
  OS preferred names (i386/amd64, i686/x86_64).
* For version strings like `5.0.0-alpha3` we use dashes in all filenames. The
  only exception is the RPM metadata (not the filename) where we replace the
  dash with an underscore (`5.0.0_alpha3`).
* We omit the release number from the filenames. It's always `1` in the metadata.

For example, here are the artifacts created for Filebeat:

```
filebeat-5.0.0-amd64.deb
filebeat-5.0.0-darwin-x86_64.tar.gz
filebeat-5.0.0-i386.deb
filebeat-5.0.0-i686.rpm
filebeat-5.0.0-linux-x86.tar.gz
filebeat-5.0.0-linux-x86_64.tar.gz
filebeat-5.0.0-windows-x86.zip
filebeat-5.0.0-windows-x86_64.zip
filebeat-5.0.0-x86_64.rpm
```

And the SNAPSHOT versions:

```
filebeat-5.0.0-SNAPSHOT-amd64.deb
filebeat-5.0.0-SNAPSHOT-darwin-x86_64.tar.gz
filebeat-5.0.0-SNAPSHOT-i386.deb
filebeat-5.0.0-SNAPSHOT-i686.rpm
filebeat-5.0.0-SNAPSHOT-linux-x86.tar.gz
filebeat-5.0.0-SNAPSHOT-linux-x86_64.tar.gz
filebeat-5.0.0-SNAPSHOT-windows-x86.zip
filebeat-5.0.0-SNAPSHOT-windows-x86_64.zip
filebeat-5.0.0-SNAPSHOT-x86_64.rpm
```
