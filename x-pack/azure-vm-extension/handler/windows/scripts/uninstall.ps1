$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

# for status
$name = "Uninstall elastic agent"
$firstOperation = "unenrolling elastic agent"
$secondOperation = "uninstalling elastic agent and removing any elastic agent related folders"
$message = "Uninstall elastic agent"
$subName = "Elastic Agent"

$serviceName = 'elastic agent'

function Uninstall-ElasticAgent {
    $INSTALL_LOCATION="C:\Program Files"
    $retries = 3
    $retryCount = 0
    $completed = $false
    while (-not $completed) {
        Try {
            $powershellVersion = Get-PowershellVersion
            $kibanaUrl = Get-Kibana-URL $powershellVersion
            if (-Not $kibanaUrl) {
                throw "Kibana url could not be found"
            }
            $password = Get-Password $powershellVersion
            $base64Auth = Get-Base64Auth $powershellVersion
            if (-Not $password -And -Not $base64Auth) {
                throw "Password  or base64auto key could not be found"
            }
            $agentId=Get-Agent-Id "$INSTALL_LOCATION\Elastic\Agent\fleet.yml"
            if (-Not $agentId) {
                throw "Agent Id could not be found"
            }
            Write-Log "Unenroll elastic agent" "INFO"
            $headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
            $headers.Add("kbn-xsrf", "true")
            #cred
            $encodedCredentials = ""
            if ($password) {
                $username = Get-Username $powershellVersion
                if (-Not $username) {
                    throw "Username could not be found"
                }
                $pair = "$($username):$($password)"
                $encodedCredentials = [System.Convert]::ToBase64String([System.Text.Encoding]::ASCII.GetBytes($pair))
            } else {
                $encodedCredentials = $base64Auth
            }
            $headers.Add('Authorization', "Basic $encodedCredentials")
            $body=(@{'force' = $true} | ConvertTo-Json)
            if ( $powershellVersion -le 3 ) {
                [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
            }else {
                $headers.Add("Accept","application/json")
            }
            try {
                $jsonResult = Invoke-WebRequest -Uri "$($kibanaUrl)/api/fleet/agents/$($agentId)/unenroll" -Body $body  -Method 'POST' -Headers $headers -UseBasicParsing -ContentType 'application/json; charset=utf-8'
            } catch {
                $jsonResult = ConvertFrom-Json $result.ErrorDetails.Message  | Select-Object
            }
            if ($jsonResult.statuscode -eq '200')
            {
                Write-Log "Unenrollment succeeded" "INFO"
            }
            elseif ($jsonResult.statuscode -eq '404' -And $jsonResult.error -eq "Not Found" ) {
                Write-Log "Elastic agent was previously unenrolled" "INFO"
            }
            else {
                throw "Unenrolling the agent failed, api request returned status $jsonResult.statuscode"
            }
            Write-Status "$name" "$firstOperation" "transitioning" "$message" "$subName" "success" "Elastic Agent service has been unenrolled"
            Write-Log "Uninstalling Elastic Agent" "INFO"
            & "$INSTALL_LOCATION\Elastic\Agent\elastic-agent.exe" uninstall --force
            Write-Log "Elastic Agent has been uninstalled" "INFO"
            Write-Log "removing directories" "INFO"
            Remove-Item "$INSTALL_LOCATION\Elastic\Agent" -Recurse -Force
            Remove-Item "$INSTALL_LOCATION\Elastic-Agent" -Recurse -Force
            Write-Log "elastic agent directories removed" "INFO"
            Write-Status "$name" "$secondOperation" "success" "$message" "$subName" "success" "Elastic Agent service has been uninstalled"
            $completed = $true
        }
        Catch {
            if ($retryCount -ge $retries) {
                Write-Log "Elastic Agent installation failed after 3 retries" "ERROR"
                Write-Log $_ "ERROR"
                Write-Log $_.ScriptStackTrace "ERROR"
                Write-Status "$name" "$firstOperation" "error" "$message" "$subName" "error" "Elastic Agent service has been uninstalled"
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


If (Get-Service $serviceName -ErrorAction SilentlyContinue) {
    Uninstall-ElasticAgent
} Else {
    Write-Log "Elastic Agent has been previously uninstalled. Cannot be found as a service." "INFO"
    Write-Status "$name" "$secondOperation" "success" "$message" "$subName" "success" "Elastic Agent service has been uninstalled"
}

