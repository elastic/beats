#!/bin/bash

res=$(wget -q -O - http://0.0.0.0:8222/streaming/channelsz | sed -n 's/"count": \([[:digit:]]\+\),/\1/p')

if [[ $res -gt 0 ]]; then
	exit 0
fi

exit 1
