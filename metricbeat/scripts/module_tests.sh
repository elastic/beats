#!/usr/bin/env bash

BEAT_PATH=github.com/elastic/beats/metricbeat

cd module

for d in * ; do
    if [[ -d $d ]]; then

        # Create names from directory name
        MODULE=$(echo $d | tr 'a-z' 'A-Z')
        VAR_NAME=${MODULE}_PORT
        declare "${VAR_NAME}"=0

        # Load part from env file
        source $d/_meta/env || true
        PORT=${!VAR_NAME}

        # TODO: How to load env variables like mysql password?

        echo $d:$PORT

        # Only modules with a port are expected to have system tests
        if [ "$PORT" -ne "0" ]; then
            cd ..

            export "${VAR_NAME}"="12345"
            # Using docker compose to wait for healthy container
            MODULE=$d PORT=${PORT} docker-compose -f module/docker-compose.yml up -d
            source module/$d/_meta/env || true; go test -tags=integration ${BEAT_PATH}/module/${d}/... -v
            MODULE=$d PORT=${PORT} docker-compose -f module/docker-compose.yml down
            cd module

            #docker build -t ${d}:metricbeat ../module/${d}/_meta/
            #docker run -p ${PORT}:${PORT} -d --name metricbeat-${MODULE} ${d}:metricbeat
            #go test -tags=integration ${BEAT_PATH}/module/${d}/... -v
            #docker stop  metricbeat-${MODULE}
            #docker rm --name metricbeat-${MODULE}
        fi;

    fi;
done


