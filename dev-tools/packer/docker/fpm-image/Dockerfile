FROM ubuntu:14.04

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# install fpm
RUN \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        build-essential ruby-dev rpm zip dos2unix libgmp3-dev && \
    gem install fpm
