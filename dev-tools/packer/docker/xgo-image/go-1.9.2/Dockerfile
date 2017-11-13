# Go cross compiler (xgo): Go 1.9.2 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-base

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Configure the root Go distribution and bootstrap based on it
RUN \
  export ROOT_DIST="https://storage.googleapis.com/golang/go1.9.2.linux-amd64.tar.gz" && \
  export ROOT_DIST_SHA1="94c889e039e3d2e94ed95e8f8cb747c5bc1c2b58" && \
  \
  $BOOTSTRAP_PURE
