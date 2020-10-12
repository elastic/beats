@echo off

REM Windows wrapper for Mage (https://magefile.org/) that installs it
REM to %GOPATH%\bin from the Beats vendor directory.
REM
REM After running this once you may invoke mage.exe directly.

WHERE mage
IF %ERRORLEVEL% NEQ 0 go get github.com/magefile/mage

mage %*
