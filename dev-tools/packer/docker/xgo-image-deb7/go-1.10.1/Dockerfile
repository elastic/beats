# Go cross compiler (xgo): Go 1.10.1 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-deb7-base

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Configure the root Go distribution and bootstrap based on it
RUN \
  export ROOT_DIST="https://storage.googleapis.com/golang/go1.10.1.linux-amd64.tar.gz" && \
  export ROOT_DIST_SHA1="0b5b278a1d785812e864158ca0bf84056cb9303c" && \
  \
  $BOOTSTRAP_PURE
