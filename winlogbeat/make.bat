@echo off

REM
REM Batch script to build and test on Windows. You can use this in conjunction
REM with the Vagrant machine.
REM

echo Building
go build
if %errorlevel% neq 0 exit /b %errorlevel%

echo Testing
go test ./...
if %errorlevel% neq 0 exit /b %errorlevel%

echo System Testing
go test -c -covermode=atomic -coverpkg ./...
if %errorlevel% neq 0 exit /b %errorlevel%
nosetests -w tests\system --process-timeout=30
if %errorlevel% neq 0 exit /b %errorlevel%
