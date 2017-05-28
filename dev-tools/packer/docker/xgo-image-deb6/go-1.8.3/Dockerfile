# Go cross compiler (xgo): Go 1.8.3 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-deb6-base

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Configure the root Go distribution and bootstrap based on it
RUN \
  export ROOT_DIST="https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz" && \
  export ROOT_DIST_SHA1="838c415896ef5ecd395dfabde5e7e6f8ac943c8e" && \
  \
  $BOOTSTRAP_PURE
