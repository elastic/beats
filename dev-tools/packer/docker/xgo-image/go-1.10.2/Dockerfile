# Go cross compiler (xgo): Go 1.10.2 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-base

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Configure the root Go distribution and bootstrap based on it
RUN \
  export ROOT_DIST="https://storage.googleapis.com/golang/go1.10.2.linux-amd64.tar.gz" && \
  export ROOT_DIST_SHA1="77f0ac48994946c0d3349baf2d973ed6decb15ca" && \
  \
  $BOOTSTRAP_PURE
