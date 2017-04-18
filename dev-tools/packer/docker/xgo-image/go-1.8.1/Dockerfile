# Go cross compiler (xgo): Go 1.8.1 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-base

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Configure the root Go distribution and bootstrap based on it
RUN \
  export ROOT_DIST="https://storage.googleapis.com/golang/go1.8.1.linux-amd64.tar.gz" && \
  export ROOT_DIST_SHA1="a433f76c569055ff8536d796995518dd91a9fa5b" && \
  \
  $BOOTSTRAP_PURE
