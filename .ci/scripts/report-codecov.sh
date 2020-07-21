#!/usr/bin/env bash
set -exuo pipefail

CODECOV_URL=https://codecov.io/bash
if [ -e /usr/local/bin/bash_standard_lib.sh ] ; then
    # shellcheck disable=SC1091
    source /usr/local/bin/bash_standard_lib.sh
    (retry 3 curl -sSLo codecov ${CODECOV_URL})
else
    curl -sSLo codecov ${CODECOV_URL}
fi

for i in "$@" ; do
    FILE="${i}/build/coverage/full.cov"
    if [ -f "${FILE}" ]; then
        bash codecov -f "${FILE}"
    fi
done
