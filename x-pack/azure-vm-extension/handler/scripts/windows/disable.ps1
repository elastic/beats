$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)


function disable-elastic-agent {
try {
    Write-Log "Stopping Elastic Agent" "INFO"
    Stop-Service "elastic agent"
    Write-Log "Elastic Agent has been stopped" "INFO"
}
catch{
Write-Log "An error occurred:" "ERROR"
        Write-Log $_ "ERROR"
        Write-Log $_.ScriptStackTrace "ERROR"
}

disable-elastic-agent
