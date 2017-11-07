@echo off

REM
REM Batch script to build and test on Windows. You can use this in conjunction
REM with the Vagrant machine.
REM

go install github.com/elastic/beats/vendor/github.com/pierrre/gotestcover
if %errorlevel% neq 0 exit /b %errorlevel%

echo Building
go build
if %errorlevel% neq 0 exit /b %errorlevel%

echo Testing
mkdir build\coverage
gotestcover -race -coverprofile=build/coverage/integration.cov github.com/elastic/beats/winlogbeat/...
if %errorlevel% neq 0 exit /b %errorlevel%

echo System Testing
go test -c -covermode=atomic -coverpkg ./...
if %errorlevel% neq 0 exit /b %errorlevel%
nosetests -v -w tests\system --process-timeout=30
if %errorlevel% neq 0 exit /b %errorlevel%

echo Aggregating Coverage Reports
python ..\dev-tools\aggregate_coverage.py -o build\coverage\system.cov .\build\system-tests\run
if %errorlevel% neq 0 exit /b %errorlevel%
python ..\dev-tools\aggregate_coverage.py -o build\coverage\full.cov .\build\coverage
if %errorlevel% neq 0 exit /b %errorlevel%
go tool cover -html=build\coverage\full.cov -o build\coverage\full.html
if %errorlevel% neq 0 exit /b %errorlevel%

echo Success
