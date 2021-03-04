$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

# for status
$name = "Disable elastic agent"
$operation = "stopping elastic agent"
$message= "Disable elastic agent"
$subName = "Elastic Agent"

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
            Write-Status "$name" "$operation" "success" "$message" "$subName" "success" "Elastic Agent service has stopped" 3
           }
        Catch {
            if ($retryCount -ge $retries) {
               Write-Log "Stopping the Elastic Agent failed after 3 retries" "ERROR"
               Write-Log $_ "ERROR"
               Write-Log $_.ScriptStackTrace "ERROR"
               Write-Status "$name" "$operation" "error" "$message" "$subName" "error" "Elastic Agent service has not stopped" 3
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
