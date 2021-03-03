$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

function Enable-ElasticAgent {
    $retries = 3
    $retryCount = 0
    $completed = $false
    while (-not $completed) {
        Try {
            Write-Log "Starting the elastic agent" "INFO"
            Start-Service "elastic agent"
            Write-Log "The elastic agent is started" "INFO"
            $completed = $true
           }
        Catch {
            if ($retryCount -ge $retries) {
               Write-Log "Starting the Elastic Agent failed after 3 retries" "ERROR"
               Write-Log $_ "ERROR"
               Write-Log $_.ScriptStackTrace "ERROR"
               exit 1
            } else {
               Write-Log "Starting the Elastic Agent has failed. retrying in 20s" "ERROR"
               Write-Log $_ "ERROR"
               Write-Log $_.ScriptStackTrace "ERROR"
               sleep 20
               $retryCount++
            }
        }
    }
}

Enable-ElasticAgent




