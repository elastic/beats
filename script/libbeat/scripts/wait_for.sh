#!/bin/bash
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
