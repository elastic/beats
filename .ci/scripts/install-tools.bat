set GOPATH=%WORKSPACE%
set MAGEFILE_CACHE=%WORKSPACE%\.magefile
set PATH=%WORKSPACE%\bin;C:\ProgramData\chocolatey\bin;%PATH%

where /q curl
IF ERRORLEVEL 1 (
 choco install curl -y --no-progress --skipdownloadcache
)
mkdir %WORKSPACE%\bin
where /q gvm
IF ERRORLEVEL 1 (
 curl -sL -o %WORKSPACE%\bin\gvm.exe https://github.com/andrewkroh/gvm/releases/download/v0.2.1/gvm-windows-amd64.exe
)
FOR /f "tokens=*" %%i IN ('"gvm.exe" use %GO_VERSION% --format=batch') DO %%i
go install -mod=vendor github.com/magefile/mage
