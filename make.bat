REM Batch script to build and test on Windows. You can use this in conjunction
REM with the Vagrant machine.
godep go build
godep go test ./...
godep go test -c -cover -covermode=count -coverpkg ./...
cd tests\system
nosetests
