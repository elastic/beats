 docker build -f windows.dockerfile --tag vm_extension_windows .
 docker run --env STACK_VERSION=7.10.2 `
 --name vm_extension_windows_run --rm -i -t vm_extension_windows `
 powershell -noexit -nologo -noprofile -executionpolicy bypass 'C://sln/handler/scripts/windows/install.ps1'
