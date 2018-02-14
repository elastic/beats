# Go cross compiler (xgo): Go 1.9.4 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-deb6-base

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Configure the root Go distribution and bootstrap based on it
RUN \
  export ROOT_DIST="https://storage.googleapis.com/golang/go1.9.4.linux-amd64.tar.gz" && \
  export ROOT_DIST_SHA1="ed1bd37c356338a5a04923c183931a96687f202e" && \
  \
  $BOOTSTRAP_PURE
