# Go cross compiler (xgo): Base cross-compilation layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM debian:7

MAINTAINER Tudor Golubenco <tudor@elastic.co>

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
    gcc-multilib \
    clang llvm-dev  libtool libxml2-dev uuid-dev libssl-dev pkg-config \
    patch make xz-utils cpio wget unzip git mercurial bzr rsync --no-install-recommends

# Inject the Go package downloader and tool-chain bootstrapper
ADD bootstrap.sh /bootstrap.sh
ENV BOOTSTRAP /bootstrap.sh
RUN chmod +x $BOOTSTRAP

# Inject the new Go root distribution downloader and secondary bootstrapper
ADD bootstrap_pure.sh /bootstrap_pure.sh
ENV BOOTSTRAP_PURE /bootstrap_pure.sh
RUN chmod +x $BOOTSTRAP_PURE

# Inject the C dependency cross compiler
ADD build_deps.sh /build_deps.sh
ENV BUILD_DEPS /build_deps.sh
RUN chmod +x $BUILD_DEPS

# Inject the container entry point, the build script
ADD build.sh /build.sh
ENV BUILD /build.sh
RUN chmod +x $BUILD

ENTRYPOINT ["/build.sh"]
