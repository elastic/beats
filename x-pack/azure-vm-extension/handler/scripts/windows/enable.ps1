. .\log.ps1
. .\helper.ps1


    #$USERNAME = ''
    #$PASSWORD = ''
     #$CLOUD_ID


# Enroll Elastic Agent
function es_agent_enroll {
  try {
    Write-Log "Starting retrieving KIBANA_URL based on $CLOUD_ID" "INFO"
    $cloud_hash=$CLOUD_ID.split(":")[-1]
    $cloud_tokens=[System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($cloud_hash))
    $cloud_elem=$cloud_tokens.split("$")
    $host_port= $cloud_elem[0]
    $ELASTICSEARCH_URL="https://$($cloud_elem[1]).$(${host_port})"
    $KIBANA_URL="https://$($cloud_elem[2]).$(${host_port})"
    Write-Log "Found Elasticsearch cluster url $ELASTICSEARCH_URL and Kibana url $KIBANA_URL" "INFO"

    $headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
    $headers.Add("Accept","Application/Json")
    $headers.Add("kbn-xsrf", "true")
    $pair = "$($USERNAME):$($PASSWORD)"
    $encodedCredentials = [System.Convert]::ToBase64String([System.Text.Encoding]::ASCII.GetBytes($pair))
    $headers.Add('Authorization', "Basic $encodedCredentials")
    $jsonResult = Invoke-WebRequest -Uri "$($KIBANA_URL)/api/fleet/enrollment-api-keys"  -Method 'GET' -Headers $headers
if ($jsonResult.statuscode -eq '200') {
    $keyValue= ConvertFrom-Json $jsonResult.Content | Select-Object -expand "list"
    Write-Log "Found enrollment_token $keyValue.id" "INFO"
    .\elastic-agent.exe install -f --kibana-url=$KIBANA_URL --enrollment-token=$keyValue
}else {

}
    }
  catch {
        Write-Log "An error occurred:" "ERROR"
        Write-Log $_ "ERROR"
        Write-Log $_.ScriptStackTrace "ERROR"
      }
}

es_agent_enroll


