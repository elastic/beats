$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

function enable-elastic-agent {
try {
    Write-Log "Starting the elastic agent" "INFO"
Start-Service elastic-agent
Write-Log "The elastic agent is started" "INFO"
}
catch{
Write-Log "An error occurred:" "ERROR"
        Write-Log $_ "ERROR"
        Write-Log $_.ScriptStackTrace "ERROR"
}

enable-elastic-agent




