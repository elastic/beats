#!/bin/sh

set -e

BASEDIR=$(dirname "$0")
ARCHDIR=${BASEDIR}/../../

runid=dashboards

cat ${ARCHDIR}/version.yml > ${BUILD_DIR}/settings-$runid.yml
gotpl ${BASEDIR}/run.sh.j2 < ${BUILD_DIR}/settings-$runid.yml > ${BUILD_DIR}/run-$runid.sh
chmod +x ${BUILD_DIR}/run-$runid.sh

docker run --rm -v ${BUILD_DIR}:/build -v ${UPLOAD_DIR}:/upload \
    -e BUILDID=$BUILDID -e SNAPSHOT=$SNAPSHOT -e RUNID=$runid -e BEAT_NAME=$BEAT_NAME \
    tudorg/fpm /build/run-$runid.sh

rm ${BUILD_DIR}/settings-$runid.yml ${BUILD_DIR}/run-$runid.sh
