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

set GVM_BIN=gvm
SET GVM_URL=https://github.com/andrewkroh/gvm/releases/download/v0.2.2
IF EXIST "%PROGRAMFILES(X86)%" (
    SET GVM_FILE=gvm-windows-amd64.exe
) ELSE (
    REM Windows 7 workers got a broken gvm installation.
    SET GVM_FILE=gvm-windows-386.exe
    set GVM_BIN=gvm.exe
    curl -L -o %WORKSPACE%\bin\%GVM_BIN% %GVM_URL%/%GVM_FILE%
)

where /q %GVM_BIN%
IF ERRORLEVEL 1 (
    set GVM_BIN=gvm.exe
    curl -L -o %WORKSPACE%\bin\%GVM_BIN% %GVM_URL%/%GVM_FILE%
    IF ERRORLEVEL 1 (
        REM The download of gvm has failed.
        exit /b 1
    )
    if EXIST %WORKSPACE%\bin\%GVM_BIN% (
        %GVM_BIN% version
    ) else (
        REM gvm.exe has not been installed for some unknown reasons
        exit /b 1
    )
)

REM Install the given go version
%GVM_BIN% --debug install %GO_VERSION%

REM Configure the given go version
FOR /f "tokens=*" %%i IN ('"%GVM_BIN%" use %GO_VERSION% --format=batch') DO %%i

go env
IF ERRORLEVEL 1 (
    REM Fallback the go installation with choco, since gvm in some workers doesn't install go correctly
    choco install golang -y -r --no-progress --version=%GO_VERSION%
    IF NOT ERRORLEVEL 0 (
        exit /b 1
    )
    refreshenv
    go env
)

go get github.com/magefile/mage
where mage
mage -version
IF ERRORLEVEL 1 (
    exit /b 1
)

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
