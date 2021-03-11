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
    $OS_SUFFIX="-windows-x86_64"
    $ALGORITHM="512"
    $INSTALL_LOCATION="C:\Program Files"
    $retries = 3
    $retryCount = 0
    $completed = $false
    $enrollment_token=""
    while (-not $completed) {
    Try {
      $powershellVersion = Get-PowershellVersion
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
      Write-Log "Starting download of elastic agent package with version $STACK_VERSION" "INFO"
      DownloadFile -Params @{'Uri'="$DOWNLOAD_URL";'OutFile'="$SAVEDFILE"}
      # write status
      Write-Status "$name" "$firstOperation" "transitioning" "$message" "$subName" "success" "Elastic Agent package has been downloaded" 1
      Write-Log "Unzip elastic agent archive" "INFO"
      Expand-Archive -LiteralPath $SAVEDFILE -DestinationPath $INSTALL_LOCATION -Force
      Write-Log "Elastic agent unzipped location $INSTALL_LOCATION" "INFO"
      Write-Log "Rename folder ..."
      Rename-Item -Path "$INSTALL_LOCATION\$INSTALL" -NewName "Elastic-Agent" -Force
      Write-Log "Folder $INSTALL renamed to 'Agent'"
      Write-Log "Start retrieving KIBANA_URL" "INFO"
      $kibana_url = Get-Kibana-URL $powershellVersion
      if (-Not $kibana_url) {
        throw "Kibana url could not be found"
        }
      $password = Get-Password $powershellVersion
      $base64Auth = Get-Base64Auth $powershellVersion
      if (-Not $password -And -Not $base64Auth) {
        throw "Password  or base64auto key could not be found"
      }
      Write-Log "Found Kibana url $kibana_url" "INFO"
      $headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
      $headers.Add("Accept","Application/Json")
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
      $jsonResult = Invoke-WebRequest -Uri "$($kibana_url)/api/fleet/enrollment-api-keys"  -Method 'GET' -Headers $headers -UseBasicParsing
      if ($jsonResult.statuscode -eq '200') {
      $keyValue= ConvertFrom-Json $jsonResult.Content | Select-Object -expand "list"
      $DEFAULT_POLICY = Get-Default-Policy $keyValue
      if (-Not $DEFAULT_POLICY) {
        Write-Log "No active Default policy has been found, will select the first active policy instead" "WARN"
        $DEFAULT_POLICY = Get-AnyActive-Policy $keyValue
      }
      if (-Not $DEFAULT_POLICY) {
        throw "No active policies were found. Please create a policy in Kibana Fleet"
      }
      Write-Log "Found enrollment_token id $DEFAULT_POLICY" "INFO"
      $jsonResult = Invoke-WebRequest -Uri "$($kibana_url)/api/fleet/enrollment-api-keys/$($DEFAULT_POLICY)"  -Method 'GET' -Headers $headers -UseBasicParsing
      if ($jsonResult.statuscode -eq '200') {
          $keyValue= ConvertFrom-Json $jsonResult.Content | Select-Object -expand "item"
          $enrollment_token=$keyValue.api_key
          Write-Log "Found enrollment_token $enrollment_token" "INFO"
          Write-Log "Installing Elastic Agent and enrolling to Fleet $kibana_url" "INFO"
          & "$INSTALL_LOCATION\Elastic-Agent\elastic-agent.exe" install -f --kibana-url=$kibana_url --enrollment-token=$enrollment_token
          Write-Log "Elastic Agent has been enrolled" "INFO"
      }else {
          throw "Retrieving the enrollment tokens has failed, api request returned status $jsonResult.statuscode"
        }
      } else {
          throw "Retrieving the enrollment token id has failed, api request returned status $jsonResult.statuscode"
      }
      $completed = $true
      # write status for both install and enroll
      Write-Status "$name" "$firstOperation" "success" "$message" "$subName" "success" "Elastic Agent has been installed" 1
      Write-Status "$name" "$secondOperation" "success" "$message" "$subName" "success" "Elastic Agent has been enrolled" 1
    }
    Catch {
      if ($retryCount -ge $retries) {
        Write-Log "Elastic Agent installation failed after 3 retries" "ERROR"
        Write-Log $_ "ERROR"
        Write-Log $_.ScriptStackTrace "ERROR"
        # write status for fail
        Write-Status "$name" "$firstOperation" "error" "$message" "$subName" "error" "Elastic Agent has not been installed" 1
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

