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
