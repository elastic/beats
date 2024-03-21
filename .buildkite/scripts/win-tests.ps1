$ErrorActionPreference = "Stop" # set -e

# Forcing to checkout again all the files with a correct autocrlf.
# Doing this here because we cannot set git clone options before.
function fixCRLF {
    Write-Host "-- Fixing CRLF in git checkout --"
    git config core.autocrlf false
    git rm --quiet --cached -r .
    git reset --quiet --hard
}

fixCRLF

$ErrorActionPreference = "Continue" # set +e
Set-Location -Path $env:BEATS_PROJECT_NAME

mage build unitTest

$EXITCODE=$LASTEXITCODE
$ErrorActionPreference = "Stop"

Exit $EXITCODE
