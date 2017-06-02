# Go cross compiler (xgo): Go 1.7.6 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-base

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Configure the root Go distribution and bootstrap based on it
RUN \
  export ROOT_DIST="https://storage.googleapis.com/golang/go1.7.6.linux-amd64.tar.gz" && \
  export ROOT_DIST_SHA1="6a7014f34048d95ab60247814a1b8b98018810ff" && \
  \
  $BOOTSTRAP_PURE
