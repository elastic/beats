set GOPATH=%WORKSPACE%
set MAGEFILE_CACHE=%WORKSPACE%\.magefile

set PATH=%WORKSPACE%\bin;C:\ProgramData\chocolatey\bin;%PATH%

echo "Upgrade chocolatey to latest version"
choco upgrade chocolatey -y

curl --version >nul 2>&1 && (
    echo found curl
) || (
    choco install curl -y --no-progress
)

mkdir %WORKSPACE%\bin

IF EXIST "%PROGRAMFILES(X86)%" (
    REM Force the gvm installation.
    SET GVM_BIN=gvm.exe
    curl -L -o %WORKSPACE%\bin\gvm.exe https://github.com/andrewkroh/gvm/releases/download/v0.3.0/gvm-windows-amd64.exe
    IF ERRORLEVEL 1 (
        REM gvm installation has failed.
        del bin\gvm.exe /s /f /q
        exit /b 1
    )
) ELSE (
    REM Windows 7 workers got a broken gvm installation.
    curl -L -o %WORKSPACE%\bin\gvm.exe https://github.com/andrewkroh/gvm/releases/download/v0.3.0/gvm-windows-386.exe
    IF ERRORLEVEL 1 (
        REM gvm installation has failed.
        del bin\gvm.exe /s /f /q
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
    REM go is not configured correctly.
    rmdir %WORKSPACE%\.gvm /s /q
    exit /b 1
)

where mage
mage -version
IF ERRORLEVEL 1 (
    go get github.com/magefile/mage
    IF ERRORLEVEL 1 (
        exit /b 1
    )
)
