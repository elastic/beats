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

REM If 32 bits then install the GVM accordingly
IF NOT EXIST "%PROGRAMFILES(X86)%" (
    curl -sL -o %WORKSPACE%\bin\gvm.exe https://github.com/andrewkroh/gvm/releases/download/v0.2.2/gvm-windows-386.exe
)

where /q gvm
IF ERRORLEVEL 1 (
    IF EXIST "%PROGRAMFILES(X86)%" (
        curl -sL -o %WORKSPACE%\bin\gvm.exe https://github.com/andrewkroh/gvm/releases/download/v0.2.2/gvm-windows-amd64.exe
    ) ELSE (
        curl -sL -o %WORKSPACE%\bin\gvm.exe https://github.com/andrewkroh/gvm/releases/download/v0.2.2/gvm-windows-386.exe
    )
    IF ERRORLEVEL 1 (
        exit /b 1
    )
    dir "%WORKSPACE%\bin" /b
)
FOR /f "tokens=*" %%i IN ('"gvm.exe" use %GO_VERSION% --format=batch') DO %%i

go env
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
