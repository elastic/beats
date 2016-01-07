#!/bin/sh

set -e

# executed from the top directory
runid=centos-$BEAT-$ARCH
cat beats/$BEAT.yml archs/$ARCH.yml version.yml > build/settings-$runid.yml
gotpl platforms/centos/run.sh.j2 < build/settings-$runid.yml > build/run-$runid.sh
gotpl platforms/centos/init.j2 < build/settings-$runid.yml > build/$runid.init
gotpl platforms/centos/systemd.j2 < build/settings-$runid.yml > build/$runid.service
chmod +x build/run-$runid.sh

docker run -v `pwd`/build:/build -e BUILDID=$BUILDID -e RUNID=$runid --name build-image tudorg/fpm /build/run-$runid.sh
docker cp build-image:/build/upload `pwd`/build/binary
docker rm -v build-image

rm build/settings-$runid.yml build/run-$runid.sh
