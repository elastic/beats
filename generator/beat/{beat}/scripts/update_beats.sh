#!/usr/bin/env bash

set -ex

BEATS_VERSION="${BEATS_VERSION:-master}"

# Find basedir and change to it
DIRNAME=$(dirname "$0")
BASEDIR=${DIRNAME}/../_beats
mkdir -p $BASEDIR
pushd $BASEDIR

# Check out beats repo for updating
GIT_CLONE=repo

# uncomment if you want to delete the cloned repository of beats
# trap "{ set +e;popd 2>/dev/null;set -e;rm -rf ${BASEDIR}/${GIT_CLONE}; }" EXIT

if [ ! -d "${GIT_CLONE}" ]; then
    git clone  -b ${BEATS_VERSION} https://github.com/elastic/beats.git ${GIT_CLONE}
else
    (
        cd ${GIT_CLONE}
        git checkout ${BEATS_VERSION}
        git reset --hard
        git pull
    )
fi

# sync
rsync -crpv --delete \
    --exclude=dev-tools/packer/readme.md.j2 \
    --include="dev-tools/***" \
    --include="script/***" \
    --include="testing/***" \
    --include="libbeat/" \
    --include=libbeat/Makefile \
    --include="libbeat/_meta/***" \
    --exclude="libbeat/_meta/fields.generated.yml" \
    --include="libbeat/docs/" \
    --include=libbeat/docs/version.asciidoc \
    --include="libbeat/processors/" \
    --include="libbeat/processors/*/" \
    --include="libbeat/processors/*/_meta/***" \
    --include="libbeat/scripts/***" \
    --include="libbeat/testing/***" \
    --include="libbeat/tests/" \
    --include="libbeat/tests/system" \
    --include=libbeat/tests/system/requirements.txt \
    --include="libbeat/tests/system/beat/***" \
    --exclude="libbeat/*" \
    --include=.go-version \
    --exclude="*" \
    ${GIT_CLONE}/ .

popd

# use exactly the same beats revision rather than $BEATS_VERSION
BEATS_REVISION=$(GIT_DIR=${BASEDIR}/${GIT_CLONE}/.git git rev-parse HEAD)
${DIRNAME}/update_govendor_deps.py ${BEATS_REVISION}
