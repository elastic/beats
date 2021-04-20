#!/usr/bin/env bash
set -euo pipefail
MSG="parameter missing."
VERSION=${1:?$MSG}

OS=$(uname -s| tr '[:upper:]' '[:lower:]')

if [ "${OS}" == "darwin" ] ; then
	SED="sed -i .bck"
else
	SED="sed -i"
fi

echo "Update stack with version ${VERSION}"
${SED} -E -e "s#(image: docker\.elastic\.co/.*):[0-9]+\.[0-9]+\.[0-9]+(-[a-f0-9]{8})?#\1:${VERSION}#g" testing/environments/snapshot-oss.yml
${SED} -E -e "s#(image: docker\.elastic\.co/.*):[0-9]+\.[0-9]+\.[0-9]+(-[a-f0-9]{8})?#\1:${VERSION}#g" testing/environments/snapshot.yml
