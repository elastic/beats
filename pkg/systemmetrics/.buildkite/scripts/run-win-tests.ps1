$ErrorActionPreference = "Stop" # set -e
# Forcing to checkout again all the files with a correct autocrlf.
# Doing this here because we cannot set git clone options before.
function fixCRLF {
    Write-Host "-- Fixing CRLF in git checkout --"
    git config core.autocrlf input
    git rm --quiet --cached -r .
    git reset --quiet --hard
}

# Note we explicitly set GOBIN to GOROOT\bin as GOROOT\bin is a part of env:PATH
# and the default go install location when GOBIN is not set is GOPATH\bin which may not be in env:PATH
function withGolang($version) {
    Write-Host "-- Install golang --"
    [Net.ServicePointManager]::SecurityProtocol = "tls12"
    Invoke-WebRequest -URI https://github.com/andrewkroh/gvm/releases/download/v0.6.0/gvm-windows-amd64.exe -Outfile C:\Windows\System32\gvm.exe
    gvm --format=powershell $version | Invoke-Expression
    go version
    go env -w GOBIN="$(go env GOROOT)\bin"
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

gotestsum --format testname --junitfile junit-win-report.xml -- -v ./...

$EXITCODE=$LASTEXITCODE
$ErrorActionPreference = "Stop"

Exit $EXITCODE
