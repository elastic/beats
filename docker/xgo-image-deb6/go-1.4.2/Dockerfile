# Go cross compiler (xgo): Go 1.4.2 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-deb6-base

MAINTAINER Péter Szilágyi <peterke@gmail.com>

# Configure the Go packages and bootstrap them
RUN \
  export DIST_LINUX_64=https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz && \
  export DIST_LINUX_64_SHA1=5020af94b52b65cc9b6f11d50a67e4bae07b0aff && \
  \
  export DIST_LINUX_32=https://storage.googleapis.com/golang/go1.4.2.linux-386.tar.gz && \
  export DIST_LINUX_32_SHA1=50557248e89b6e38d395fda93b2f96b2b860a26a && \
  \
  $BOOTSTRAP
