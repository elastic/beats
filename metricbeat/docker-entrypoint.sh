#!/usr/bin/env bash
set -e

# Wait for. Params: host, port, service
waitFor() {
    echo -n "Waiting for ${3}(${1}:${2}) to start."
    for ((i=1; i<=90; i++)) do
        if nc -vz ${1} ${2} 2>/dev/null; then
            echo
            echo "${3} is ready!"
            return 0
        fi

        ((i++))
        echo -n '.'
        sleep 1
    done

    echo
    echo >&2 "${3} is not available"
    echo >&2 "Address: ${1}:${2}"
}

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
