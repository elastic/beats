$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

# for status
$name = "Install elastic agent"
$firstOperation = "installing elastic agent"
$secondOperation = "enrolling elastic agent"
$message= "Install elastic agent"
$subName = "Elastic Agent"

function Install-ElasticAgent {
    $INSTALL_LOCATION="C:\Program Files"
    $retries = 3
    $retryCount = 0
    $completed = $false
    while (-not $completed) {
        Try {
            $powershellVersion = Get-PowershellVersion
            $kibana_url = Get-Kibana-URL $powershellVersion
            if (-Not $kibana_url) {
                throw "Kibana url could not be found"
                }
            $username = Get-Username $powershellVersion
            if (-Not $username) {
                throw "Username could not be found"
                }
            $password = Get-Password $powershellVersion
            if (-Not $password) {
                throw "Password could not be found"
                }
            $agentId=Get-Agent-Id "$INSTALL_LOCATION\Elastic\Agent\fleet.yml"
            if (-Not $agentId) {
                throw "Agent Id could not be found"
                }
            Write-Log "Unenroll elastic agent" "INFO"
            $headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
            $headers.Add("Accept","application/json")
            $headers.Add("kbn-xsrf", "true")
            $pair = "$($username):$($password)"
            $encodedCredentials = [System.Convert]::ToBase64String([System.Text.Encoding]::ASCII.GetBytes($pair))
            $headers.Add('Authorization', "Basic $encodedCredentials")
            $body=(@{'force' = $true} | ConvertTo-Json)
            $jsonResult = Invoke-WebRequest -Uri "$($kibana_url)/api/fleet/agents/$($agentId)/unenroll" -Body $body  -Method 'POST' -Headers $headers -UseBasicParsing -ContentType 'application/json; charset=utf-8'
            if ($jsonResult.statuscode -eq '200') {
                Write-Log "Unenrollment succeeded" "INFO"
            } else {
                throw "Unenrolling the agent failed, api request returned status $jsonResult.statuscode"
                }
            Write-Log "Uninstalling Elastic Agent" "INFO"
            & "$INSTALL_LOCATION\Elastic\Agent\elastic-agent.exe" uninstall --force
            Write-Log "Elastic Agent has been uninstalled" "INFO"
            Write-Log "removing directories" "INFO"
            Remove-Item "$INSTALL_LOCATION\Elastic\Agent" -Recurse -Force
            Remove-Item "$INSTALL_LOCATION\Elastic-Agent" -Recurse -Force
            Write-Log "elastic agent directories removed" "INFO"
            $completed = $true
        }
        Catch {
          if ($retryCount -ge $retries) {
            Write-Log "Elastic Agent installation failed after 3 retries" "ERROR"
            Write-Log $_ "ERROR"
            Write-Log $_.ScriptStackTrace "ERROR"
            exit 1
          } else {
            Write-Log "Elastic Agent installation failed. retrying in 20s" "ERROR"
            Write-Log $_ "ERROR"
            Write-Log $_.ScriptStackTrace "ERROR"
            sleep 20
            $retryCount++
            }
        }
    }
}

Install-ElasticAgent
