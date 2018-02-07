FROM ubuntu:16.04

MAINTAINER Tudor Golubenco <tudor@elastic.co>

# install fpm
RUN \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        autoconf build-essential libffi-dev ruby-dev rpm zip dos2unix libgmp3-dev

RUN gem install fpm -v 1.9.2
