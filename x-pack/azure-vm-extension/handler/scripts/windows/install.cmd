set ES_EXT_DIR=%~dp0

echo %ES_EXT_DIR%

powershell -nologo -noprofile -executionpolicy unrestricted Import-Module %ES_EXT_DIR%\install.ps1;Run-Powershell2-With-Dot-Net4
