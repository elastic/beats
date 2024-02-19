$ErrorActionPreference = "Stop" # set -e
$WorkFolder = $env:BEATS_PROJECT_NAME

# Forcing to checkout again all the files with a correct autocrlf.
# Doing this here because we cannot set git clone options before.
function fixCRLF {
    Write-Host "-- Fixing CRLF in git checkout --"
    git config core.autocrlf false
    git rm --quiet --cached -r .
    git reset --quiet --hard
}
# function withChoco {
#     Write-Host "-- Configure Choco --"
#     $env:ChocolateyInstall = Convert-Path "$((Get-Command choco).Path)\..\.."
#     Import-Module "$env:ChocolateyInstall\helpers\chocolateyProfile.psm1"
# }
function withGolang($version) {
    Write-Host "-- Installing Go $version --"
    $goDownloadPath = Join-Path $env:TEMP "go_installer.msi"
    $goInstallerUrl = "https://golang.org/dl/go$version.windows-amd64.msi"
    Invoke-WebRequest -Uri $goInstallerUrl -OutFile $goDownloadPath
    Start-Process -FilePath "msiexec.exe" -ArgumentList "/i $goDownloadPath /quiet" -Wait
    $env:GOPATH = "${env:ProgramFiles}\Go"
    $env:GOBIN = "${env:GOPATH}\bin"
    $env:Path += ";$env:GOPATH;$env:GOBIN"
    go version
    installGoDependencies
}

function withPython($version) {
    Write-Host "-- Installing Python $version --"
    [Net.ServicePointManager]::SecurityProtocol = "tls11, tls12, ssl3"
    $pyDownloadPath = Join-Path $env:TEMP "python-$version-amd64.exe"
    $pyInstallerUrl = "https://www.python.org/ftp/python/$version/python-$version-amd64.exe"
    Invoke-WebRequest -UseBasicParsing -Uri $pyInstallerUrl -OutFile $pyDownloadPath
    Start-Process -FilePath $pyDownloadPath -ArgumentList "/quiet", "InstallAllUsers=1", "PrependPath=1", "Include_test=0" -Wait
    $pyBinPath = "${env:ProgramFiles}\Python311"
    $env:Path += ";$pyBinPath"
    python --version
}

function withMinGW {
    Write-Host "-- Installing MinGW --"
    $gwInstallerUrl = "https://github.com/brechtsanders/winlibs_mingw/releases/download/12.1.0-14.0.6-10.0.0-ucrt-r3/winlibs-x86_64-posix-seh-gcc-12.1.0-llvm-14.0.6-mingw-w64ucrt-10.0.0-r3.zip"
    $gwDownloadPath = "$env:TEMP\winlibs-x86_64.zip"
    Invoke-WebRequest -Uri $gwInstallerUrl -OutFile $gwDownloadPath
    Expand-Archive -Path $gwDownloadPath -DestinationPath "$env:TEMP"
    $gwBinPath = "$env:TEMP\mingw64\bin"
    $env:Path += ";$gwBinPath"
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

function withNmap($version) {
    Write-Host "-- Installing Nmap $version --"
    $nmapInstallerUrl = "https://nmap.org/dist/nmap-$version-setup.exe"
    $nmapDownloadPath = "$env:TEMP\nmap-$version-setup.exe"
    Invoke-WebRequest -UseBasicParsing -Uri $nmapInstallerUrl -OutFile $nmapDownloadPath
    Start-Process -FilePath $nmapDownloadPath -ArgumentList "/S" -Wait
}

fixCRLF

# withChoco

withGolang $env:GO_VERSION

withPython $env:SETUP_WIN_PYTHON_VERSION

withMinGW

withNmap $env:NMAP_WIN_VERSION

$ErrorActionPreference = "Continue" # set +e

Set-Location -Path $WorkFolder

$magefile = ".magefile"
$env:MAGEFILE_CACHE = $magefile

New-Item -ItemType Directory -Force -Path "build"

if ($env:BUILDKITE_PIPELINE_SLUG -eq "beats-xpack-libbeat") {
    mage -w reader/etw build goUnitTest
} else {
    mage build unitTest
}

$EXITCODE=$LASTEXITCODE
$ErrorActionPreference = "Stop"

Exit $EXITCODE
