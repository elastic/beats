# Go cross compiler (xgo): Go 1.7 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-deb6-base

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Configure the root Go distribution and bootstrap based on it
RUN \
  export ROOT_DIST="https://storage.googleapis.com/golang/go1.7.linux-amd64.tar.gz" && \
  export ROOT_DIST_SHA1="a744e29da97fc3aadad1ee0d7d89b0d899645e50" && \
  \
  $BOOTSTRAP_PURE
