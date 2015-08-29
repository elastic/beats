#!/bin/sh

set -e

# executed from the top directory
runid=windows-$BEAT-$RELEASE-$ARCH

cat beats/$BEAT.yml archs/$ARCH.yml releases/$RELEASE.yml > build/settings-$runid.yml
gotpl platforms/windows/run.sh.j2 < build/settings-$runid.yml > build/run-$runid.sh
gotpl platforms/windows/install-service.ps1.j2 < build/settings-$runid.yml > build/install-service-$BEAT.ps1
gotpl platforms/windows/uninstall-service.ps1.j2 < build/settings-$runid.yml > build/uninstall-service-$BEAT.ps1
chmod +x build/run-$runid.sh
docker run -v `pwd`/build:/build -e BUILDID=$BUILDID -e RUNID=$runid tudorg/fpm /build/run-$runid.sh
rm build/settings-$runid.yml build/run-$runid.sh
