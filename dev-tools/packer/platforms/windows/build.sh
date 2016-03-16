#!/bin/sh

set -e

BASEDIR=$(dirname "$0")
ARCHDIR=${BASEDIR}/../..

# executed from the top directory
runid=windows-$BEAT-$ARCH

cat beats/$BEAT.yml ${ARCHDIR}/archs/$ARCH.yml version.yml > build/settings-$runid.yml
gotpl ${BASEDIR}/run.sh.j2 < build/settings-$runid.yml > build/run-$runid.sh
gotpl ${BASEDIR}/install-service.ps1.j2 < build/settings-$runid.yml > build/install-service-$BEAT.ps1
gotpl ${BASEDIR}/uninstall-service.ps1.j2 < build/settings-$runid.yml > build/uninstall-service-$BEAT.ps1
chmod +x build/run-$runid.sh

docker run -v `pwd`/build:/build -e BUILDID=$BUILDID -e RUNID=$runid --name build-image tudorg/fpm /build/run-$runid.sh
docker cp build-image:/build/upload `pwd`/build/binary
docker rm -v build-image

rm build/settings-$runid.yml build/run-$runid.sh
