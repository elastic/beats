$env:GO_VERSION = Get-Content -Path .go-version

# Install gvm and go
# TODO: Move GVM download to the base VM image
$env:GvmVersion = "0.5.2"
[Net.ServicePointManager]::SecurityProtocol = "tls12"
$env:GoVersion = Get-Content -Path .go-version
Invoke-WebRequest -URI https://github.com/andrewkroh/gvm/releases/download/v$env:GvmVersion/gvm-windows-amd64.exe -Outfile C:\Windows\System32\gvm.exe
gvm --format=powershell $env:GO_VERSION | Invoke-Expression
go version

$GOPATH = $(go env GOPATH)
$env:Path = "$GOPATH\bin;" + $env:Path
[Environment]::SetEnvironmentVariable("GOPATH", "$GOPATH", [EnvironmentVariableTarget]::Machine)
[Environment]::SetEnvironmentVariable("Path", "$GOPATH\bin;$env:Path", [EnvironmentVariableTarget]::Machine)
