#!/usr/bin/env bash
#
# This script is executed after the release snapshot stage.
# to help with the debugging/troubleshooting
#
set -uo pipefail
set +e
if [ -f release-manager-report.out ] ; then
    echo "There are some errors, let's guess what they are about:" > release-manager-report.txt
    if grep 'Vault responded with HTTP status code' release-manager-report.out ; then
        echo 'Environmental issue with Vault. Try again' >> release-manager-report.txt
    fi
    if grep 'Cannot write to file' release-manager-report.out ; then
        echo 'Artifacts were not generated. Likely a genuine issue' >> release-manager-report.txt
    fi
    if grep 'does not exist' release-manager-report.out ; then
        echo 'Build file does not exist in the unified release. Likely the branch is not supported yet. Contact the release platform team' >> release-manager-report.txt
    fi
else
    echo 'WARN: release-manager-report.out does not exist'
fi
