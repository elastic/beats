set -e
set -m

/entrypoint.sh couchbase-server &

sleep 1

waitForCouchbase() {
    echo -n "Waiting for Couchbase to start."
    for ((i=1; i<=90; i++)) do
        if nc -vz 127.0.0.1 8091 2>/dev/null; then
            return 0
        fi

        ((i++))
        echo -n '.'
        sleep 1
    done

    echo
    echo >&2 "Failed to Start Couchbase"
}

waitForCouchbase

# Setup index and memory quota
curl -v -X POST http://127.0.0.1:8091/pools/default -d memoryQuota=300 -d indexMemoryQuota=300

# Setup services
curl -v http://127.0.0.1:8091/node/controller/setupServices -d services=kv%2Cn1ql%2Cindex

# Setup credentials
curl -v http://127.0.0.1:8091/settings/web -d port=8091 -d username=Administrator -d password=password

# Load travel-sample bucket
curl -v -u Administrator:password -X POST http://127.0.0.1:8091/sampleBuckets/install -d '["beer-sample"]'

fg 1
