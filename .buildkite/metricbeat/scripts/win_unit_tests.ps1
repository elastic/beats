$ErrorActionPreference = "Stop" # set -e
$WorkFolder = "metricbeats"
# Forcing to checkout again all the files with a correct autocrlf.
# Doing this here because we cannot set git clone options before.
function fixCRLF {
    Write-Host "-- Fixing CRLF in git checkout --"
    git config core.autocrlf input
    git rm --quiet --cached -r .
    git reset --quiet --hard
}
function withGolang($version) {
    Write-Host "-- Install golang --"
    choco install -y golang --version $version
    $env:ChocolateyInstall = Convert-Path "$((Get-Command choco).Path)\..\.."
    Import-Module "$env:ChocolateyInstall\helpers\chocolateyProfile.psm1"
    refreshenv
    go version
    go env
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
withGolang $env:GO_VERSION
installGoDependencies

$ErrorActionPreference = "Continue" # set +e

Push-Location "$WorkFolder"

mage build unitTest

Pop-Location

$EXITCODE=$LASTEXITCODE
$ErrorActionPreference = "Stop"

Exit $EXITCODE
