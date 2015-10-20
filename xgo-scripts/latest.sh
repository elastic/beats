#!/bin/bash

FILES=$(find `pwd`/build/binary/* -type f)
echo $BUILDID

for currentfile in ${FILES}; do
	latestfile=${currentfile/$BUILDID/latest}
	cp -f $currentfile $latestfile

	# Replace buildid through latest in sha1 file
	if [ `echo $latestfile | grep -c "sha1" ` -gt 0 ]; then
		sed -i.bak -e s/$BUILDID/latest/g $latestfile
		rm $latestfile.bak
	fi
done
