# Windows

This module uses `.evtx` files as a source to generate the test data. 
In order to add a new `.evtx` samples you will need to re-generate the
sample data. 

Example command to generate sample data for PowerShell.

```sh
cd powershell
go test -v ./... -update
```

This will update the golden files used from testing.