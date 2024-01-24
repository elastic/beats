$ErrorActionPreference = "Stop" # set -e
$WorkFolder = "metricbeat"
# Forcing to checkout again all the files with a correct autocrlf.
# Doing this here because we cannot set git clone options before.
function fixCRLF {
    Write-Host "-- Fixing CRLF in git checkout --"
    git config core.autocrlf false
    git rm --quiet --cached -r .
    git reset --quiet --hard
}
function withChoco {
    Write-Host "-- Configure Choco --"
    $env:ChocolateyInstall = Convert-Path "$((Get-Command choco).Path)\..\.."
    Import-Module "$env:ChocolateyInstall\helpers\chocolateyProfile.psm1"
}
function withGolang($version) {
    Write-Host "-- Install golang $version --"
    choco install -y golang --version=$version
    refreshenv
    go version
}
function withPython($version) {
    Write-Host "-- Install Python $version --"
    choco install python --version=$version
    refreshenv
    python --version
}
function withMinGW {
    Write-Host "-- Install MinGW --"
    choco install mingw -y
    refreshenv
}
function installGoDependencies {
    $installPackages = @(
        "github.com/magefile/mage"
        "github.com/elastic/go-licenser"
        "golang.org/x/tools/cmd/goimports"
        "github.com/jstemmer/go-junit-report/v2"
        "gotest.tools/gotestsum"
    )
    foreach ($pkg in $installPackages) {
        go install "$pkg@latest"
    }
}

fixCRLF

withChoco

withGolang $env:GO_VERSION

installGoDependencies

withPython $env:SETUP_WIN_PYTHON_VERSION

withMinGW

$ErrorActionPreference = "Continue" # set +e

Push-Location $WorkFolder

New-Item -ItemType Directory -Force -Path "build"
mage build unitTest

Pop-Location

$EXITCODE=$LASTEXITCODE
$ErrorActionPreference = "Stop"

Exit $EXITCODE
