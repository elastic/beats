#!/bin/sh

set -e

# executed from the top directory
runid=debian-$BEAT-$RELEASE-$ARCH
cat beats/$BEAT.yml archs/$ARCH.yml releases/$RELEASE.yml > build/settings-$runid.yml
j2 --format yaml platforms/debian/run.sh.j2 < build/settings-$runid.yml > build/run-$runid.sh
chmod +x build/run-$runid.sh
docker run -v `pwd`/build:/build -e BUILDID=$BUILDID tudorg/fpm /build/run-$runid.sh
rm build/settings-$runid.yml build/run-$runid.sh
