#!/usr/bin/env bash
set -e

# Wait for. Params: host, port, service
waitFor() {

    if [ $# == "3" ]; then
        SERVICE=$3
        ADDRESS="${1}:${2}"
    else
        SERVICE=$2
        ADDRESS="${1}" 
    fi

    echo -n "Waiting for ${SERVICE}(${ADDRESS}) to start."

    for ((i=1; i<=90; i++)) do

        if [ $# == "3" ]; then
            if nc -vz ${1} ${2} 2>/dev/null; then
                echo
                echo "${SERVICE} is ready!"
                return 0
            fi
        else
            if nc -Uvz ${1} 2>/dev/null; then
                echo
                echo "${SERVICE} is ready!"
                return 0
            fi
        fi

        ((i++))
        echo -n '.'
        sleep 1
    done

    echo
    echo >&2 "${SERVICE} is not available"
    echo >&2 "Address: ${ADDRESS} "

    nc -U /tmp/haproxy-stats.sock
}

# Main
waitFor ${APACHE_HOST} ${APACHE_PORT} Apache
waitFor ${MYSQL_HOST} ${MYSQL_PORT} MySQL
waitFor ${NGINX_HOST} ${NGINX_PORT} Nginx
waitFor ${REDIS_HOST} ${REDIS_PORT} Redis
waitFor ${ZOOKEEPER_HOST} ${ZOOKEEPER_PORT} Zookeeper
waitFor ${HAPROXY_ADDR} HAProxy
exec "$@"
