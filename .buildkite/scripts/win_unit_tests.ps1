$ErrorActionPreference = "Stop" # set -e
if ($env:BEATS_PROJECT_NAME) {
    if ($env:BEATS_PROJECT_NAME -like "*x-pack/*") {
        $WorkFolder = $env:BEATS_PROJECT_NAME -replace "/", "\"
    } else {
        $WorkFolder = $env:BEATS_PROJECT_NAME
    }
}
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
    $downloadPath = Join-Path $env:TEMP "go_installer.msi"
    $goInstallerUrl = "https://golang.org/dl/go$version.windows-amd64.msi"
    Invoke-WebRequest -Uri $goInstallerUrl -OutFile $downloadPath
    Start-Process -FilePath "msiexec.exe" -ArgumentList "/i $downloadPath /quiet" -Wait
    $goBinPath = "${env:ProgramFiles}\Go\bin"
    $env:Path += ";$goBinPath"
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

function withNpcap($version) {
    $npcapDownloadPath = Join-Path $env:TEMP "npcap-$version.exe"
    $npcapInstallerUrl = "https://npcap.com/dist/npcap-$version.exe"
    Invoke-WebRequest -Uri $npcapInstallerUrl -OutFile $npcapDownloadPath
    Start-Process -FilePath "$env:TEMP\npcap-$version.exe" -ArgumentList "/S" -Wait
}

fixCRLF

withChoco

withGolang $env:GO_VERSION

installGoDependencies

withPython $env:SETUP_WIN_PYTHON_VERSION

withMinGW

if ($env:BUILDKITE_PIPELINE_SLUG -eq "beats-packetbeat") {
    withNpcap
}

$ErrorActionPreference = "Continue" # set +e

Push-Location $WorkFolder

New-Item -ItemType Directory -Force -Path "build"

if ($env:BUILDKITE_PIPELINE_SLUG -eq "beats-xpack-libbeat") {
    mage -w reader/etw build goUnitTest
} else {
    mage build unitTest
}

Pop-Location

$EXITCODE=$LASTEXITCODE
$ErrorActionPreference = "Stop"

Exit $EXITCODE
