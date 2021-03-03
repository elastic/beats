$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

function Disable-ElasticAgent {
    $retries = 3
    $retryCount = 0
    $completed = $false
    while (-not $completed) {
        Try {
            Write-Log "Stopping Elastic Agent" "INFO"
            Stop-Service "elastic agent"
            Write-Log "Elastic Agent has been stopped" "INFO"
            $completed = $true
           }
        Catch {
            if ($retryCount -ge $retries) {
               Write-Log "Stopping the Elastic Agent failed after 3 retries" "ERROR"
               Write-Log $_ "ERROR"
               Write-Log $_.ScriptStackTrace "ERROR"
               exit 1
            } else {
               Write-Log "Stopping the Elastic Agent failed. retrying in 20s" "ERROR"
               Write-Log $_ "ERROR"
               Write-Log $_.ScriptStackTrace "ERROR"
               sleep 20
               $retryCount++
            }
        }
    }
}

Disable-ElasticAgent
