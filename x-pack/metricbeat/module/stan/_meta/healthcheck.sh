#!/bin/bash

res=$(wget -q -O - http://0.0.0.0:8222/streaming/channelsz | sed -n 's/"count": \([[:digit:]]\+\),/\1/p')

if [[ $res = 0 ]]; then
	exit 1
fi

exit 0
