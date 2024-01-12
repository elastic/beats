$ErrorActionPreference = "Stop" # set -e

# Forcing to checkout again all the files with a correct autocrlf.
# Doing this here because we cannot set git clone options before.
function fixCRLF() {
    Write-Host "-- Fixing CRLF in git checkout --"
    git config core.autocrlf false
    git rm --quiet --cached -r .
    git reset --quiet --hard
}

function withGolang() {
    #    Write-Host "-- Install golang $env:GOLANG_VERSION --"
    #    choco install golang -y --version $env:GOLANG_VERSION

    Write-Host "-- Install golang 1.20.7 --"
    choco install golang -y --version 1.20.7

    $choco = Convert-Path "$((Get-Command choco).Path)\..\.."
    Import-Module "$choco\helpers\chocolateyProfile.psm1"
    refreshenv
    go version
    go env
}

function installGoDependencies() {
    $installPackages = @(
        "github.com/magefile/mage"
        "github.com/elastic/go-licenser"
        "golang.org/x/tools/cmd/goimports"
        "github.com/jstemmer/go-junit-report"
        "github.com/tebeka/go2xunit"
    )
    foreach ($pkg in $installPackages) {
        go install "$pkg"
    }
}

fixCRLF

$ErrorActionPreference = "Continue" # set +e

Set-Location -Path filebeat
New-Item -ItemType Directory -Force -Path "build"
withGolang
installGoDependencies

mage build unitTest

$EXITCODE=$LASTEXITCODE
$ErrorActionPreference = "Stop"

Exit $EXITCODE
