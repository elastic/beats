# Go cross compiler (xgo): Go 1.7.1 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-base

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Configure the root Go distribution and bootstrap based on it
RUN \
  export ROOT_DIST="https://storage.googleapis.com/golang/go1.7.1.linux-amd64.tar.gz" && \
  export ROOT_DIST_SHA1="919ab01305ada0078a9fdf8a12bb56fb0b8a1444" && \
  \
  $BOOTSTRAP_PURE
