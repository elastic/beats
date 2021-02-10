$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

Write-Log "Uninstalling Elastic Agent" "INFO"
 $INSTALL_LOCATION="C:\Program Files"
 #Start-Process powershell.exe -Verb RunAs -ArgumentList ('-noprofile -noexit  -elevated' -f "$INSTALL_LOCATION\Elastic\Agent\elastic-agent.exe uninstall")

 & "$INSTALL_LOCATION\Elastic\Agent\elastic-agent.exe" uninstall
Write-Log "Elastic Agent has been uninstalled" "INFO"
