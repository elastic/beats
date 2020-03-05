#!/usr/bin/env bash
set -exuo pipefail

MSG="parameter missing."
GO_VERSION=${GO_VERSION:?$MSG}
PROPERTIES_FILE=${PROPERTIES_FILE:-"go_env.properties"}
HOME=${HOME:?$MSG}
ARCH=$(uname -s| tr '[:upper:]' '[:lower:]')
GVM_CMD="${HOME}/bin/gvm"

mkdir -p "${HOME}/bin"

curl -sSLo "${GVM_CMD}" "https://github.com/andrewkroh/gvm/releases/download/v0.2.1/gvm-${ARCH}-amd64"
chmod +x "${GVM_CMD}"

gvm ${GO_VERSION}|cut -d ' ' -f 2|tr -d '\"' > ${PROPERTIES_FILE}

eval $(gvm ${GO_VERSION})
GO111MODULE=off go get -u github.com/kardianos/govendor

docker images
cd metricbeat
docker tag store/oracle/database-instantclient:12.2.0.1 database-instantclient:12.2.0.1
TESTING_ENVIRONMENT=snapshot ES_BEATS=.. docker-compose -p metricbeat21e89f892431fca6dc2c9c17fa89dd8a9b03e92a  -f docker-compose.yml build --force-rm
