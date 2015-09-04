# Go cross compiler (xgo): Go 1.4.2 layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM tudorg/xgo-base

MAINTAINER Péter Szilágyi <peterke@gmail.com>

# Configure the Go packages and bootstrap them
RUN \
  export DIST_LINUX_64=https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz && \
  export DIST_LINUX_64_SHA1=5020af94b52b65cc9b6f11d50a67e4bae07b0aff && \
  \
  export DIST_LINUX_32=https://storage.googleapis.com/golang/go1.4.2.linux-386.tar.gz && \
  export DIST_LINUX_32_SHA1=50557248e89b6e38d395fda93b2f96b2b860a26a && \
  \
  export DIST_LINUX_ARM=http://dave.cheney.net/paste/go1.4.2.linux-arm~armv5-1.tar.gz && \
  export DIST_LINUX_ARM_SHA1=1bcfc8ef9c2aa381722b71b8c8d83cb58e973116 && \
  \
  export DIST_OSX_64=https://storage.googleapis.com/golang/go1.4.2.darwin-amd64-osx10.6.tar.gz && \
  export DIST_OSX_64_SHA1=00c3f9a03daff818b2132ac31d57f054925c60e7 && \
  \
  export DIST_OSX_32=https://storage.googleapis.com/golang/go1.4.2.darwin-386-osx10.6.tar.gz && \
  export DIST_OSX_32_SHA1=fb3e6b30f4e1b1be47bbb98d79dd53da8dec24ec && \
  \
  export DIST_WIN_64=https://storage.googleapis.com/golang/go1.4.2.windows-amd64.zip && \
  export DIST_WIN_64_SHA1=91b229a3ff0a1ce6e791c832b0b4670bfc5457b5 && \
  \
  export DIST_WIN_32=https://storage.googleapis.com/golang/go1.4.2.windows-386.zip && \
  export DIST_WIN_32_SHA1=0e074e66a7816561d7947ff5c3514be96f347dc4 && \
  \
  $BOOTSTRAP
