#!/usr/bin/env bash
set -e

source ./../libbeat/scripts/wait_for.sh

# Main
waitFor ${APACHE_HOST} ${APACHE_PORT} Apache
waitFor ${CEPH_HOST} ${CEPH_PORT} Ceph
waitFor ${COUCHBASE_HOST} ${COUCHBASE_PORT} Couchbase
waitFor ${HAPROXY_HOST} ${HAPROXY_PORT} HAProxy
waitFor ${KAFKA_HOST} ${KAFKA_PORT} Kafka
waitFor ${MONGODB_HOST} ${MONGODB_PORT} MongoDB
waitFor ${MYSQL_HOST} ${MYSQL_PORT} MySQL
waitFor ${NGINX_HOST} ${NGINX_PORT} Nginx
waitFor ${PHPFPM_HOST} ${PHPFPM_PORT} PHP_FPM
waitFor ${POSTGRESQL_HOST} ${POSTGRESQL_PORT} Postgresql
waitFor ${PROMETHEUS_HOST} ${PROMETHEUS_PORT} Prometheus
waitFor ${REDIS_HOST} ${REDIS_PORT} Redis
waitFor ${ZOOKEEPER_HOST} ${ZOOKEEPER_PORT} Zookeeper
exec "$@"
