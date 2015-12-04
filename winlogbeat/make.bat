REM Batch script to build and test on Windows. You can use this in conjunction
REM with the Vagrant machine.
REM
REM If running inside of Vagrant do this first:
REM mkdir C:\Gopath\src\github.com\elastic
REM mklink /d C:\Gopath\src\github.com\elastic\winlogbeat \\vboxsvr\vagrant

REM This is already done inside the Vagrant box.
REM set PATH=%PATH%;%GOPATH%\bin

go build
go test -race ./...

REM Coverage report:
REM godep go test -c -cover -coverpkg ./...

go test -c -covermode=atomic -coverpkg ./...
nosetests -w tests\system --process-timeout=30
