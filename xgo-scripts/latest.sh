#!/bin/bash

FILES=$(find `pwd`/build/binary/* -type f)
echo $BUILDID

for currentfile in ${FILES}; do
	latestfile=${currentfile/$BUILDID/latest}
	cp -f $currentfile $latestfile
done
