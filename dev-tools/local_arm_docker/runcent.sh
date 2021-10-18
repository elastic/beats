#!/bin/sh
docker run --rm -it --user root --platform linux/arm64/v8 --entrypoint=/bin/bash -i --volume=$HOME/projects/beats/:/beats docker.elastic.co/beats/elastic-agent-complete:$1-arm64  
