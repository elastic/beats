$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

Write-Log "Stopping Elastic Agent" "INFO"
Stop-Service "elastic agent"
Write-Log "Elastic Agent has been stopped" "INFO"
