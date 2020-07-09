set GOPATH=%WORKSPACE%
set MAGEFILE_CACHE=%WORKSPACE%\.magefile
set PATH=%WORKSPACE%\bin;C:\ProgramData\chocolatey\bin;%PATH%

REM Configure GCC for either 32 or 64 bits
IF EXIST "%PROGRAMFILES(X86)%" (
    set PATH=C:\tools\mingw64\bin;%PATH%
) ELSE (
    set PATH=C:\tools\mingw32\bin;%PATH%
)

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
)
FOR /f "tokens=*" %%i IN ('"gvm.exe" use %GO_VERSION% --format=batch') DO %%i

go env
go get github.com/magefile/mage
mage -version
where mage

IF not exist C:\Python38\python.exe (
    REM Install python 3.8.
    choco install python -y -r --no-progress --version 3.8.2 || echo ERROR && exit /b
)
python --version
where python

where /q gcc
IF ERRORLEVEL 1 (
    REM Install mingw 5.3.0
    choco install mingw -y -r --no-progress --version 5.3.0 || echo ERROR && exit /b
)
gcc --version
where gcc
