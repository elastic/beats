$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

$INSTALL_LOCATION="C:\Program Files"

function unenroll-elastic-agent {
& $INSTALL_LOCATION\Elastic-Agent\elastic-agent.exe inspect
[string[]]$fileContent = Get-Content "$INSTALL_LOCATION\Elastic\Agent\fleet.yml"
$content = ''
foreach ($line in $fileContent) { $content = $content + "`n" + $line }
$yaml = ConvertFrom-YAML $content
echo $yaml

Write-Log "Unenroll elastic agent" "INFO"
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("Accept","Application/Json")
$headers.Add("kbn-xsrf", "true")
$pair = "$($username):$($password)"
$encodedCredentials = [System.Convert]::ToBase64String([System.Text.Encoding]::ASCII.GetBytes($pair))
$headers.Add('Authorization', "Basic $encodedCredentials")
$jsonResult = Invoke-WebRequest -Uri "$($kibana_url)/api/fleet/agents/${enrollmentResponse.item.id}/unenroll"  -Method 'POST' -Headers $headers -UseBasicParsing
if ($jsonResult.statuscode -eq '200') {
$keyValue= ConvertFrom-Json $jsonResult.Content | Select-Object -expand "item"
$enrollment_token=$keyValue.api_key
Write-Log "Found enrollment_token $enrollment_token" "INFO"
Write-Log "Installing Elastic Agent and enrolling to Fleet $kibana_url" "INFO"
}else {

      }
}


function uninstall-elastic-agent {
try {
    Write-Log "Uninstalling Elastic Agent" "INFO"
    $INSTALL_LOCATION="C:\Program Files"
 #Start-Process powershell.exe -Verb RunAs -ArgumentList ('-noprofile -noexit  -elevated' -f "$INSTALL_LOCATION\Elastic\Agent\elastic-agent.exe uninstall")
    & "$INSTALL_LOCATION\Elastic\Agent\elastic-agent.exe" uninstall
    Write-Log "Elastic Agent has been uninstalled" "INFO"
    }
catch{
    Write-Log "An error occurred:" "ERROR"
    Write-Log $_ "ERROR"
    Write-Log $_.ScriptStackTrace "ERROR"
}
}

#uninstall-elastic-agent

unenroll-elastic-agent
