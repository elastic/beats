# Go cross compiler (xgo): Base cross-compilation layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
#
# Released under the MIT license.

FROM ubuntu:14.04

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# Configure the Go environment, since it's not going to change
ENV PATH   /usr/local/go/bin:$PATH
ENV GOPATH /go


# Inject the remote file fetcher and checksum verifier
ADD fetch.sh /fetch.sh
ENV FETCH /fetch.sh
RUN chmod +x $FETCH


# Make sure apt-get is up to date and dependent packages are installed
# XXX: The first line is a workaround for the "Sum hash mismatch" error, from here:
# https://askubuntu.com/questions/760574/sudo-apt-get-update-failes-due-to-hash-sum-mismatch
RUN \
  apt-get clean && \
  apt-get update && \
  apt-get install -y automake autogen build-essential ca-certificates           \
    gcc-arm-linux-gnueabi g++-arm-linux-gnueabi libc6-dev-armel-cross           \
    gcc-multilib  g++-multilib mingw-w64 clang llvm-dev                         \
    libtool libxml2-dev uuid-dev libssl-dev swig pkg-config patch               \
    make xz-utils cpio wget zip unzip p7zip git mercurial bzr texinfo help2man  \
    binutils-multiarch rsync                                                    \
    --no-install-recommends

# Configure the container for OSX cross compilation
ENV OSX_SDK     MacOSX10.11.sdk
ENV OSX_NDK_X86 /usr/local/osx-ndk-x86

RUN \
  OSX_SDK_PATH=https://github.com/phracker/MacOSX-SDKs/releases/download/MacOSX10.11.sdk/MacOSX10.11.sdk.tar.xz && \
  $FETCH $OSX_SDK_PATH f3430e3d923644e66c0c13f7a48754e7b6aa2e3f       && \
  \
  git clone https://github.com/tpoechtrager/osxcross.git && \
  mv `basename $OSX_SDK_PATH` /osxcross/tarballs/        && \
  \
  sed -i -e 's|-march=native||g' /osxcross/build_clang.sh /osxcross/wrapper/build.sh && \
  UNATTENDED=yes OSX_VERSION_MIN=10.6 /osxcross/build.sh                             && \
  mv /osxcross/target $OSX_NDK_X86                                                   && \
  \
  rm -rf /osxcross

ADD patch.tar.xz $OSX_NDK_X86/SDK/$OSX_SDK/usr/include/c++
ENV PATH $OSX_NDK_X86/bin:$PATH


# Inject the Go package downloader and tool-chain bootstrapper
ADD bootstrap.sh /bootstrap.sh
ENV BOOTSTRAP /bootstrap.sh
RUN chmod +x $BOOTSTRAP

# Inject the new Go root distribution downloader and secondary bootstrapper
ADD bootstrap_pure.sh /bootstrap_pure.sh
ENV BOOTSTRAP_PURE /bootstrap_pure.sh
RUN chmod +x $BOOTSTRAP_PURE

# Inject the C dependency cross compiler
ADD build_deps.sh /build_deps.sh
ENV BUILD_DEPS /build_deps.sh
RUN chmod +x $BUILD_DEPS

# Inject the container entry point, the build script
ADD build.sh /build.sh
ENV BUILD /build.sh
RUN chmod +x $BUILD

ENTRYPOINT ["/build.sh"]
