$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

$INSTALL_LOCATION="C:\Program Files"

function install-elastic-agent {
    $OS_SUFFIX="-windows-x86_64"
    $ALGORITHM="512"
    $INSTALL_LOCATION="C:\Program Files"
    try {
        $STACK_VERSION= Get-Stack-Version
        if ( $STACK_VERSION -eq "" ) {
        throw "Elastic stack version could not be found"
        }
        $INSTALL= "elastic-agent-${STACK_VERSION}${OS_SUFFIX}"
        $PACKAGE="${INSTALL}.zip"
        $SHASUM="$PACKAGE.sha$ALGORITHM"
        $DOWNLOAD_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}"
        $SHASUM_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}.sha512"
        $SAVEDFILE="$env:temp\" + $PACKAGE
        Write-Log "Started" "INFO"
        DownloadFile -Params @{'Uri'="$DOWNLOAD_URL";'OutFile'="$SAVEDFILE"}
        Write-Log "Unzip elastic agent archive" "INFO"
        Expand-Archive -LiteralPath $SAVEDFILE -DestinationPath $INSTALL_LOCATION
        Write-Log "Elastic agent unzipped location $INSTALL_LOCATION" "INFO"
        Write-Log "Rename folder ..."
        Rename-Item -Path "$INSTALL_LOCATION\$INSTALL" -NewName "Elastic-Agent"
        Write-Log "Folder $INSTALL renamed to 'Agent'"
    }
    catch {
      Write-Log "An error occurred:" "ERROR"
      Write-Log $_ "ERROR"
      Write-Log $_.ScriptStackTrace "ERROR"
    }

}

# Enroll Elastic Agent
function enroll-elastic-agent {
  try {
    Write-Log "Start retrieving KIBANA_URL" "INFO"
     $powershellVersion = Get-PowershellVersion
    $kibana_url = Get-Kibana-URL $powershellVersion
    $username = Get-Username $powershellVersion
    $password = Get-Password $powershellVersion
    if ( $kibana_url -eq "") {
    throw "Kibana url could not be found"
    }
    $enrollment_token=""
    Write-Log "Found Kibana url $kibana_url" "INFO"
    $headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
    $headers.Add("Accept","Application/Json")
    $headers.Add("kbn-xsrf", "true")
    $pair = "$($username):$($password)"
    $encodedCredentials = [System.Convert]::ToBase64String([System.Text.Encoding]::ASCII.GetBytes($pair))
    $headers.Add('Authorization', "Basic $encodedCredentials")
    $jsonResult = Invoke-WebRequest -Uri "$($kibana_url)/api/fleet/enrollment-api-keys"  -Method 'GET' -Headers $headers -UseBasicParsing
    if ($jsonResult.statuscode -eq '200') {
    $keyValue= ConvertFrom-Json $jsonResult.Content | Select-Object -expand "list"
    $enrollment_token_key=$keyValue.id
    Write-Log "Found enrollment_token id $enrollment_token_key" "INFO"
     $jsonResult = Invoke-WebRequest -Uri "$($kibana_url)/api/fleet/enrollment-api-keys/$enrollment_token_key"  -Method 'GET' -Headers $headers -UseBasicParsing
     if ($jsonResult.statuscode -eq '200') {
         $keyValue= ConvertFrom-Json $jsonResult.Content | Select-Object -expand "item"
         $enrollment_token=$keyValue.api_key
         Write-Log "Found enrollment_token $enrollment_token" "INFO"
         Write-Log "Installing Elastic Agent and enrolling to Fleet $kibana_url" "INFO"
         #echo Start-Process powershell.exe -Verb RunAs -ArgumentList ('-noprofile -noexit -file "{0}" -elevated' -f ("$INSTALL_LOCATION\Elastic-Agent\elastic-agent.exe" install -f --kibana-url=$KIBANA_URL --enrollment-token=$enrollment_token))
         #Start-Process "$INSTALL_LOCATION\Elastic-Agent\elastic-agent.exe" -Verb RunAs -ArgumentList (install -f --kibana-url=$KIBANA_URL --enrollment-token=$enrollment_token) | Out-Default

         & "$INSTALL_LOCATION\Elastic-Agent\elastic-agent.exe" enroll -f --kibana-url=$kibana_url $enrollment_token
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


function install {
    install-elastic-agent
    enroll-elastic-agent
}

install
