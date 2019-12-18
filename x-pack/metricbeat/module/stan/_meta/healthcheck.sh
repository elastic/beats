#!/bin/bash

res=$(wget -q -O - http://0.0.0.0:8222/streaming/channelsz | egrep -e '"count": (\d)+,' | cut -d":" -f2 |cut -d"," -f1 | xargs)

if [[ $res = 0 ]]; then
	exit 1
fi

exit 0
