# Go cross compiler (xgo): Go 1.6 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-deb6-base

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Configure the root Go distribution and bootstrap based on it
RUN \
  export ROOT_DIST="https://storage.googleapis.com/golang/go1.6.2.linux-amd64.tar.gz" && \
  export ROOT_DIST_SHA1="b8318b09de06076d5397e6ec18ebef3b45cd315d" && \
  \
  $BOOTSTRAP_PURE
