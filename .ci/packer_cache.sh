#!/usr/bin/env bash
#
# this file is run daily to generate worker packer images
#

# shellcheck disable=SC1091
source /usr/local/bin/bash_standard_lib.sh

# shellcheck disable=SC1091
source ./dev-tools/common.bash

# Docker images used on Dockerfiles 2019-07-12
# aerospike:3.9.0
# alpine:edge
# apache/couchdb:1.7
# busybox:latest
# ceph/daemon:master-6373c6a-jewel-centos-7-x86_64
# cockroachdb/cockroach:v19.1.1
# consul:1.4.2
# coredns/coredns:1.5.0
# couchbase:4.5.1
# debian:latest
# debian:stretch
# docker.elastic.co/beats-dev/fpm:1.11.0
# docker.elastic.co/beats/metricbeat:6.5.4
# docker.elastic.co/beats/metricbeat:7.2.0
# docker.elastic.co/elasticsearch/elasticsearch:7.2.0
# docker.elastic.co/kibana/kibana:7.2.0
# docker.elastic.co/logstash/logstash:7.2.0
# docker.elastic.co/observability-ci/database-instantclient:12.2.0.1
# envoyproxy/envoy:v1.7.0
# exekias/localkube-image
# haproxy:1.8
# httpd:2.4.20
# java:8-jdk-alpine
# jplock/zookeeper:3.4.8
# maven:3.3-jdk-8
# memcached:1.4.35-alpine
# microsoft/mssql-server-linux:2017-GA
# mongo:3.4
# mysql:5.7.12
# nats:1.3.0
# nginx:1.9
# oraclelinux:7
# postgres:9.5.3
# prom/prometheus:v2.6.0
# python:3.6-alpine
# quay.io/coreos/etcd:v3.3.10
# rabbitmq:3.7.4-management
# redis:3.2.12-alpine
# redis:3.2.4-alpine
# store/oracle/database-enterprise:12.2.0.1
# traefik:1.6-alpine
# tsouza/nginx-php-fpm:php-7.1
# ubuntu:16.04
# ubuntu:trusty

get_go_version

DOCKER_IMAGES="docker.elastic.co/observability-ci/database-instantclient:12.2.0.1
docker.elastic.co/observability-ci/database-enterprise:12.2.0.1
docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-arm
docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-darwin
docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-main
docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-main-debian7
docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-main-debian8
docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-mips
docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-ppc
docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-s390x
golang:${GO_VERSION}
"
if [ -x "$(command -v docker)" ]; then
  for image in ${DOCKER_IMAGES}
  do
  (retry 2 docker pull ${image}) || echo "Error pulling ${image} Docker image, we continue"
  done

  docker tag \
    docker.elastic.co/observability-ci/database-instantclient:12.2.0.1 \
    store/oracle/database-instantclient:12.2.0.1 \
    || echo "Error setting the Oracle Instant Client tag"
  docker tag \
    docker.elastic.co/observability-ci/database-enterprise:12.2.0.1 \
    store/oracle/database-enterprise:12.2.0.1 \
    || echo "Error setting the Oracle Dtabase tag"
fi
