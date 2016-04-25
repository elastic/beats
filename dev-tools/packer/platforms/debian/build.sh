#!/bin/sh

set -e

BASEDIR=$(dirname "$0")
ARCHDIR=${BASEDIR}/../../

# executed from the top directory
runid=debian-$BEAT-$ARCH

cat beats/$BEAT.yml ${ARCHDIR}/archs/$ARCH.yml version.yml > build/settings-$runid.yml
gotpl ${BASEDIR}/run.sh.j2 < build/settings-$runid.yml > build/run-$runid.sh
chmod +x build/run-$runid.sh
gotpl ${BASEDIR}/init.j2 < build/settings-$runid.yml > build/$runid.init
gotpl ${BASEDIR}/systemd.j2 < build/settings-$runid.yml > build/$runid.service
gotpl ${BASEDIR}/beatname.sh.j2 < build/settings-$runid.yml > build/beatname-$runid.sh
chmod +x build/beatname-$runid.sh

docker run --rm -v `pwd`/build:/build \
    -e BUILDID=$BUILDID -e SNAPSHOT=$SNAPSHOT -e RUNID=$runid \
    tudorg/fpm /build/run-$runid.sh

rm build/settings-$runid.yml build/run-$runid.sh
