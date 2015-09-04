# Go cross compiler (xgo): Base cross-compilation layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM ubuntu:14.04

MAINTAINER Péter Szilágyi <peterke@gmail.com>

# Configure the Go environment, since it's not going to change
ENV PATH   /usr/local/go/bin:$PATH
ENV GOPATH /go


# Inject the remote file fetcher and checksum verifier
ADD fetch.sh /fetch.sh
ENV FETCH /fetch.sh
RUN chmod +x $FETCH


# Make sure apt-get is up to date and dependent packages are installed
RUN \
  apt-get update && \
  apt-get install -y automake autogen build-essential ca-certificates \
    gcc-arm-linux-gnueabi libc6-dev-armel-cross gcc-multilib gcc-mingw-w64 \
    clang llvm-dev  libtool libxml2-dev uuid-dev libssl-dev pkg-config \
    patch make xz-utils cpio wget unzip git mercurial bzr --no-install-recommends

# Configure the container for OSX cross compilation
ENV OSX_SDK_PATH https://github.com/trevd/android_platform_build2/raw/master/osxsdks10.6.tar.gz
ENV OSX_SDK      MacOSX10.6.sdk

RUN \
  git clone https://github.com/tpoechtrager/osxcross.git && \
  \
  $FETCH $OSX_SDK_PATH f526b4ae9806e8d31e3b094e3f004f8f160a3fad

RUN \
  tar -xzf `basename $OSX_SDK_PATH` --strip-components 1 SDKs/$OSX_SDK && \
  tar -cjf /osxcross/tarballs/$OSX_SDK.tar.bz2 $OSX_SDK && \
  rm -rf `basename $OSX_SDK_PATH` $OSX_SDK && \
  \
  UNATTENDED=yes /osxcross/build.sh
ENV PATH /osxcross/target/bin:$PATH


# Inject the Go package downloader and tool-chain bootstrapper
ADD bootstrap.sh /bootstrap.sh
ENV BOOTSTRAP /bootstrap.sh
RUN chmod +x $BOOTSTRAP

# Inject the C dependency cross compiler
ADD build_deps.sh /build_deps.sh
ENV BUILD_DEPS /build_deps.sh
RUN chmod +x $BUILD_DEPS

# Inject the container entry point, the build script
ADD build.sh /build.sh
ENV BUILD /build.sh
RUN chmod +x $BUILD

ENTRYPOINT ["/build.sh"]
