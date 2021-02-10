$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)


# Enroll Elastic Agent
function es_agent_enroll {
    #For testing only
    #$STACK_VERSION=$env:STACK_VERSION
    #$CLOUD_ID=$env:CLOUD_ID
    #$USERNAME=$env:USERNAME
    #$PASSWORD=$env:PASSWORD
    #end
  try {
    Write-Log "Start retrieving KIBANA_URL based on $CLOUD_ID" "INFO"
    $cloud_hash=$CLOUD_ID.split(":")[-1]
    $cloud_tokens=[System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($cloud_hash))
    $cloud_elem=$cloud_tokens.split("$")
    $host_port= $cloud_elem[0]
    $ELASTICSEARCH_URL="https://$($cloud_elem[1]).$(${host_port})"
    $KIBANA_URL="https://$($cloud_elem[2]).$(${host_port})"
    $enrollment_token=""
    $INSTALL_LOCATION="C:\Program Files"
    Write-Log "Found Elasticsearch cluster url $ELASTICSEARCH_URL and Kibana url $KIBANA_URL" "INFO"

    $headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
    $headers.Add("Accept","Application/Json")
    $headers.Add("kbn-xsrf", "true")
    $pair = "$($USERNAME):$($PASSWORD)"
    $encodedCredentials = [System.Convert]::ToBase64String([System.Text.Encoding]::ASCII.GetBytes($pair))
    $headers.Add('Authorization', "Basic $encodedCredentials")
    $jsonResult = Invoke-WebRequest -Uri "$($KIBANA_URL)/api/fleet/enrollment-api-keys"  -Method 'GET' -Headers $headers -UseBasicParsing
    if ($jsonResult.statuscode -eq '200') {
    $keyValue= ConvertFrom-Json $jsonResult.Content | Select-Object -expand "list"
    $enrollment_token_key=$keyValue.id
    Write-Log "Found enrollment_token id $enrollment_token_key" "INFO"
     $jsonResult = Invoke-WebRequest -Uri "$($KIBANA_URL)/api/fleet/enrollment-api-keys/$enrollment_token_key"  -Method 'GET' -Headers $headers -UseBasicParsing
     if ($jsonResult.statuscode -eq '200') {
         $keyValue= ConvertFrom-Json $jsonResult.Content | Select-Object -expand "item"
         $enrollment_token=$keyValue.api_key
         Write-Log "Found enrollment_token $enrollment_token" "INFO"
         Write-Log "Installing Elastic Agent and enrolling to Fleet $KIBANA_URL" "INFO"
         #echo Start-Process powershell.exe -Verb RunAs -ArgumentList ('-noprofile -noexit -file "{0}" -elevated' -f ("$INSTALL_LOCATION\Elastic-Agent\elastic-agent.exe" install -f --kibana-url=$KIBANA_URL --enrollment-token=$enrollment_token))
         #Start-Process "$INSTALL_LOCATION\Elastic-Agent\elastic-agent.exe" -Verb RunAs -ArgumentList (install -f --kibana-url=$KIBANA_URL --enrollment-token=$enrollment_token) | Out-Default

         & "$INSTALL_LOCATION\Elastic-Agent\elastic-agent.exe" install -f --kibana-url=$KIBANA_URL --enrollment-token=$enrollment_token
     }else {

      }
    } else { }
  }
   catch {
        Write-Log "An error occurred:" "ERROR"
        Write-Log $_ "ERROR"
        Write-Log $_.ScriptStackTrace "ERROR"
      }
}

es_agent_enroll


