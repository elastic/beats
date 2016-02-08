#!/usr/bin/env bash
set -e

waitForApache() {
    echo -n "Waiting for apache(${APACHE_HOST}:${APACHE_PORT}) to start."
    for ((i=1; i<=90; i++)) do
        if nc -vz ${APACHE_HOST} ${APACHE_PORT} 2>/dev/null; then
            echo
            echo "Apache is ready!"
            return 0
        fi

        ((i++))
        echo -n '.'
        sleep 1
    done

    echo
    echo >&2 'Apache is not available'
    echo >&2 "Address: ${APACHE_HOST}:${APACHE_PORT}"
}

waitForRedis() {
    echo -n "Waiting for redis(${REDIS_HOST}:${REDIS_PORT}) to start."
    for ((i=1; i<=90; i++)) do
        if nc -vz ${REDIS_HOST} ${REDIS_PORT} 2>/dev/null; then
            echo
            echo "Redis is ready!"
            return 0
        fi

        ((i++))
        echo -n '.'
        sleep 1
    done

    echo
    echo >&2 'Redis is not available'
    echo >&2 "Address: ${REDIS_HOST}:${REDIS_PORT}"
}


waitForMySQL() {
    echo -n "Waiting for mysql(${MYSQL_HOST}:${MYSQL_PORT}) to start."
    for ((i=1; i<=90; i++)) do
        if nc -vz ${MYSQL_HOST} ${MYSQL_PORT} 2>/dev/null; then
            echo
            echo "MYSQL_HOST is ready!"
            return 0
        fi

        ((i++))
        echo -n '.'
        sleep 1
    done

    echo
    echo >&2 'MySQL is not available'
    echo >&2 "Address: ${MYSQL_HOST}:${MYSQL_PORT}"
}

# Main
waitForApache
waitForRedis
waitForMySQL
exec "$@"
