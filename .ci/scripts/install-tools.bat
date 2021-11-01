set GOPATH=%WORKSPACE%
set MAGEFILE_CACHE=%WORKSPACE%\.magefile
set PATH=%WORKSPACE%\bin;C:\ProgramData\chocolatey\bin;%PATH%

where /q curl
IF ERRORLEVEL 1 (
 choco install curl -y --no-progress --skipdownloadcache
)
mkdir %WORKSPACE%\bin
IF EXIST "%PROGRAMFILES(X86)%" (
    REM Force the gvm installation.
    SET GVM_BIN=gvm.exe
    curl -L -o %WORKSPACE%\bin\gvm.exe https://github.com/andrewkroh/gvm/releases/download/v0.3.0/gvm-windows-amd64.exe
    IF ERRORLEVEL 1 (
        REM gvm installation has failed.
        exit /b 1
    )
) ELSE (
    REM Windows 7 workers got a broken gvm installation.
    curl -L -o %WORKSPACE%\bin\gvm.exe https://github.com/andrewkroh/gvm/releases/download/v0.3.0/gvm-windows-386.exe
    IF ERRORLEVEL 1 (
        REM gvm installation has failed.
        exit /b 1
    )
)

SET GVM_BIN=gvm.exe
WHERE /q %GVM_BIN%
%GVM_BIN% version

REM Install the given go version
%GVM_BIN% --debug install %GO_VERSION%

REM Configure the given go version
FOR /f "tokens=*" %%i IN ('"%GVM_BIN%" use %GO_VERSION% --format=batch') DO %%i

go env
FOR /f "tokens=*" %%i IN ('"gvm.exe" use %GO_VERSION% --format=batch') DO %%i

go install github.com/elastic/beats/vendor/github.com/magefile/mage
mage -version
where mage

if not exist C:\Python38\python.exe (
    REM Install python 3.8.
    choco install python -y -r --no-progress --version 3.8.2 || echo ERROR && exit /b
)
python --version
where python
