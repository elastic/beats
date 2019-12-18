#!/bin/bash

res=$(curl -s http://0.0.0.0:8222/streaming/channelsz | jq '.count')

if [[ $res = 0 ]]; then
	exit 1
fi

exit 0
