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
 curl -sL -o %WORKSPACE%\bin\gvm.exe https://github.com/andrewkroh/gvm/releases/download/v0.2.2/gvm-windows-amd64.exe
)
FOR /f "tokens=*" %%i IN ('"gvm.exe" use %GO_VERSION% --format=batch') DO %%i

go env
go get github.com/magefile/mage
mage -version
where mage

if not exist C:\Python38\python.exe (
    REM Install python 3.8.
    choco install python -y -r --no-progress --version 3.8.2 || echo ERROR && exit /b
)
python --version
where python

