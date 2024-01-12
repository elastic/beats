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

function findLog() {
  $mainDir = Get-Location
  $logFilename = "docker_corrupted.log"
  $file = Get-ChildItem -Path $mainDir -Filter $logFilename -Recurse -File | Select-Object -First 1

  Write-Host ":: LOG FILE PATH :: $($file.FullName)"
#  buildkite-agent meta-data set CORRUPTED_LOG_FILEPATH $($file.FullName)
}

function getLogLineEnding {
    [CmdletBinding()]
    param (
        [Parameter(Mandatory=$true, Position=0)]
        [string]$FilePath
    )

    $fileContent = [IO.File]::ReadAllText($FilePath)

    Write-Host ":: CHECK CRLF ::"

    if ($fileContent.Contains("`r`n")) { Write-Output "CRLF (Windows line endings)" }
    elseif ($fileContent.Contains("`n")) { Write-Output "LF (Unix line endings)" }
    else { Write-Output "Unable to determine line ending type." }
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
