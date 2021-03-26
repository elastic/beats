$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory config-update.ps1)

# for status enable
$nameE = "Enable elastic agent"
$operationE = "starting elastic agent"
$messageE = "Enable elastic agent"

# for status install
$name = "Install elastic agent"
$firstOperation = "installing elastic agent"
$secondOperation = "enrolling elastic agent"
$message = "Install elastic agent"
$subName = "Elastic Agent"


$serviceName = 'elastic agent'

function Install-ElasticAgent {
    $INSTALL_LOCATION="C:\Program Files"
    $OS_SUFFIX="-windows-x86_64"
    $ALGORITHM="512"
    $retries = 3
    $retryCount = 0
    $completed = $false
    $enrollment_token= ""
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
            Write-Status "$name" "$firstOperation" "transitioning" "$message" "$subName" "success" "Elastic Agent package has been downloaded"
            Write-Log "Unzip elastic agent archive" "INFO"
            if ( $powershellVersion -le 3 ) {
                Add-Type -Assembly "System.IO.Compression.Filesystem"
                [System.IO.Compression.ZipFile]::ExtractToDirectory($SAVEDFILE,$INSTALL_LOCATION )
            }else {
                Expand-Archive -LiteralPath $SAVEDFILE -DestinationPath $INSTALL_LOCATION -Force
            }
            Write-Log "Elastic agent unzipped location $INSTALL_LOCATION" "INFO"
            Write-Log "Rename folder ..."
            Rename-Item -Path "$INSTALL_LOCATION\$INSTALL" -NewName "Elastic-Agent" -Force
            Write-Log "Folder $INSTALL renamed to 'Agent'"
            Write-Log "Start retrieving KIBANA_URL" "INFO"
            $kibanaUrl = Get-Kibana-URL $powershellVersion
            if (-Not $kibanaUrl) {
                throw "Kibana url could not be found"
            }
            $password = Get-Password $powershellVersion
            $base64Auth = Get-Base64Auth $powershellVersion
            if (-Not $password -And -Not $base64Auth) {
                throw "Password  or base64auto key could not be found"
            }
            Write-Log "Found Kibana url $kibanaUrl" "INFO"
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
            if ( $powershellVersion -le 3 ) {
                [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
            }else {
                $headers.Add("Accept","application/json")
            }
            #enable Fleet
            $jsonResult = Invoke-WebRequest -Uri "$($kibanaUrl)/api/fleet/setup"  -Method 'POST' -Headers $headers -UseBasicParsing
            if ($jsonResult.statuscode -eq '200') {
                Write-Log "Enable Fleet is now available $jsonResult" "INFO"
                $jsonResult = Invoke-WebRequest -Uri "$($kibanaUrl)/api/fleet/agents/setup"  -Method 'POST' -Headers $headers -UseBasicParsing
                if ($jsonResult.statuscode -eq '200') {
                    Write-Log "Enable Fleet agents if now available $jsonResult" "INFO"
                }else {
                    throw "Enabling Fleet Agents failed with $jsonResult.statuscode"
                }
            }
            else {
                throw "Enabling Fleet failed with $jsonResult.statuscode"
            }
            # end enable Fleet
            $jsonResult = Invoke-WebRequest -Uri "$($kibanaUrl)/api/fleet/enrollment-api-keys"  -Method 'GET' -Headers $headers -UseBasicParsing
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
                $jsonResult = Invoke-WebRequest -Uri "$($kibanaUrl)/api/fleet/enrollment-api-keys/$($DEFAULT_POLICY)"  -Method 'GET' -Headers $headers -UseBasicParsing
                if ($jsonResult.statuscode -eq '200') {
                    $keyValue= ConvertFrom-Json $jsonResult.Content | Select-Object -expand "item"
                    $enrollment_token=$keyValue.api_key
                    Write-Log "Found enrollment_token $enrollment_token" "INFO"
                    Write-Log "Installing Elastic Agent and enrolling to Fleet $kibanaUrl" "INFO"
                    & "$INSTALL_LOCATION\Elastic-Agent\elastic-agent.exe" install -f --kibana-url=$kibanaUrl --enrollment-token=$enrollment_token
                    Write-Log "Elastic Agent has been enrolled" "INFO"
                }else {
                    throw "Retrieving the enrollment tokens has failed, api request returned status $jsonResult.statuscode"
                }
            } else {
                throw "Retrieving the enrollment token id has failed, api request returned status $jsonResult.statuscode"
            }
            Write-Log "Setting Env Variable for sequence" "INFO"
            Set-SequenceEnvVariables
            $completed = $true
            # write status for both install and enroll
            Write-Status "$name" "$firstOperation" "success" "$message" "$subName" "success" "Elastic Agent has been installed"
            Write-Status "$name" "$secondOperation" "success" "$message" "$subName" "success" "Elastic Agent has been enrolled"
        }
        Catch {
            if ($retryCount -ge $retries) {
                Write-Log "Elastic Agent installation failed after 3 retries" "ERROR"
                Write-Log $_ "ERROR"
                Write-Log $_.ScriptStackTrace "ERROR"
                # write status for fail
                Write-Status "$name" "$firstOperation" "error" "$message" "$subName" "error" "Elastic Agent has not been installed"
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

function Enable-ElasticAgent {
    $retries = 3
    $retryCount = 0
    $completed = $false
    while (-not $completed) {
        Try {
            Write-Log "Starting the elastic agent" "INFO"
            Start-Service "$serviceName"
            Write-Log "The elastic agent is started" "INFO"
            $completed = $true
            Write-Status "$nameE" "$operationE" "success" "$messageE" "$subName" "success" "Elastic Agent service has started"
           }
        Catch {
            if ($retryCount -ge $retries) {
                Write-Log "Starting the Elastic Agent failed after 3 retries" "ERROR"
                Write-Log $_ "ERROR"
                Write-Log $_.ScriptStackTrace "ERROR"
                Write-Status "$nameE" "$operationE" "error" "$messageE" "$subName" "error" "Elastic Agent service has not started"
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

function Reconfigure-ElasticAgent {
    $retries = 3
    $retryCount = 0
    $completed = $false
    while (-not $completed) {
        Try {
            Write-Log "Stopping Elastic Agent" "INFO"
            Stop-Service "elastic agent"
            Write-Log "Elastic Agent has been stopped" "INFO"
            Uninstall-Old-ElasticAgent
            Install-ElasticAgent
            $completed = $true
            Write-Status "$name" "$operationE" "success" "$message" "$subName" "success" "Elastic Agent has been reconfigured and reinstalled"
        }
        Catch {
            if ($retryCount -ge $retries) {
                Write-Log "Starting the Elastic Agent failed after 3 retries" "ERROR"
                Write-Log $_ "ERROR"
                Write-Log $_.ScriptStackTrace "ERROR"
                Write-Status "$nameE" "$operationE" "error" "$messageE" "$subName" "error" "Elastic Agent service has not been reconfigured"
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


If (Get-Service $serviceName -ErrorAction SilentlyContinue) {
    If (Is-New-Config) {
        Write-Log "New configuration file has been added. The elastic agent will reinstall" "INFO"
        Reconfigure-ElasticAgent
    }
    If ((Get-Service $serviceName).Status -eq 'Running') {
        Write-Log "Elastic Agent service is running" "INFO"
        Write-Status "$nameE" "$operationE" "success" "$messageE" "$subName" "success" "Elastic Agent service is running"
    } Else {
        Enable-ElasticAgent
    }
} Else {
    Install-ElasticAgent
    Enable-ElasticAgent
}

