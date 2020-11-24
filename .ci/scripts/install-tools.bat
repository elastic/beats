set GOPATH=%WORKSPACE%
set MAGEFILE_CACHE=%WORKSPACE%\.magefile

REM Configure GCC for either 32 or 64 bits
set MINGW_ARCH=64
IF NOT EXIST "%PROGRAMFILES(X86)%" (
    set MINGW_ARCH=32
)
set PATH=%WORKSPACE%\bin;C:\ProgramData\chocolatey\bin;C:\tools\mingw%MINGW_ARCH%\bin;%PATH%

where /q curl
IF ERRORLEVEL 1 (
    choco install curl -y --no-progress --skipdownloadcache
)
mkdir %WORKSPACE%\bin

IF EXIST "%PROGRAMFILES(X86)%" (
    REM Force the gvm installation.
    SET GVM_BIN=gvm.exe
    curl -L -o %WORKSPACE%\bin\gvm.exe https://s3.us-east-1.amazonaws.com/deploy.andrewkroh.com/gvm/gvm-windows-amd64.exe
    IF ERRORLEVEL 1 (
        REM gvm installation has failed.
        exit /b 1
    )
) ELSE (
    REM Windows 7 workers got a broken gvm installation.
    curl -L -o %WORKSPACE%\bin\gvm.exe https://github.com/andrewkroh/gvm/releases/download/v0.2.3/gvm-windows-386.exe
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
IF ERRORLEVEL 1 (
    REM go is not configured correctly let's fallback with some scripting
    where /q unzip
    IF ERRORLEVEL 1 (
        choco install unzip -y --no-progress --skipdownloadcache
    )
    IF EXIST "%PROGRAMFILES(X86)%" (
        curl -L -o %USERPROFILE%\.gvm\go.zip https://storage.googleapis.com/golang/go%GO_VERSION%.windows-amd64.zip
        unzip -q -u -o %USERPROFILE%\.gvm\go.zip -d %USERPROFILE%\.gvm\.go
        MOVE %USERPROFILE%\.gvm\versions\go%GO_VERSION%.windows.amd64 %USERPROFILE%\.gvm\versions\go%GO_VERSION%.windows.amd64.old
        MOVE /Y .go\go %USERPROFILE%\.gvm\versions\go%GO_VERSION%.windows.amd64
    ) ELSE (
        curl -L -o %USERPROFILE%\.gvm\go.zip https://storage.googleapis.com/golang/go%GO_VERSION%.windows-386.zip
        unzip -q -u -o %USERPROFILE%\.gvm\go.zip -d %USERPROFILE%\.gvm\.go
        MOVE %USERPROFILE%\.gvm\versions\go%GO_VERSION%.windows.386 %USERPROFILE%\.gvm\versions\go%GO_VERSION%.windows.386.old
        MOVE /Y .go\go %USERPROFILE%\.gvm\versions\go%GO_VERSION%.windows.386
    )
    go env
)

go get github.com/magefile/mage
where mage
mage -version
IF ERRORLEVEL 1 (
    exit /b 1
)

REM Set the USERPROFILE to the previous location to fix issues with chocolatey in windows 2019
SET PREVIOUS_USERPROFILE=%USERPROFILE%
SET USERPROFILE=%OLD_USERPROFILE%
IF NOT EXIST C:\Python38\python.exe (
    REM Install python 3.8
    choco install python -y -r --no-progress --version 3.8.5
    IF NOT ERRORLEVEL 0 (
        exit /b 1
    )
)
python --version
where python

where /q gcc
IF ERRORLEVEL 1 (
    REM Install mingw 5.3.0
    choco install mingw -y -r --no-progress --version 5.3.0
    IF NOT ERRORLEVEL 0 (
        exit /b 1
    )
)
gcc --version
where gcc

REM Reset the USERPROFILE
SET USERPROFILE=%PREVIOUS_USERPROFILE%