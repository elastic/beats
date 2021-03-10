function GetDirectory
{
  $Invocation = (Get-Variable MyInvocation -Scope 1).Value
  Split-Path $Invocation.MyCommand.Path
}

$scriptDir = GetDirectory

$extensionRoot = [System.IO.Path]::GetFullPath("$scriptDir\\..")

function Get-PowershellVersion {
  if(!$powershellVersion)
  {
      $global:powershellVersion = $PSVersionTable.PSVersion.Major
  }
  $powershellVersion
}

function Run-Powershell2-With-Dot-Net4 {
  $powershellVersion = Get-PowershellVersion

  if ( $powershellVersion -lt 3 ) {
    reg add hklm\software\microsoft\.netframework /v OnlyUseLatestCLR /t REG_DWORD /d 1 /f
    reg add hklm\software\wow6432node\microsoft\.netframework /v OnlyUseLatestCLR /t REG_DWORD /d 1 /f
  }
}

function Get-CloudId($powershellVersion) {
  $cloudId = Get-PublicSettings-From-Config-Json "cloudId"  $powershellVersion
  if ( $cloudId){
    return $cloudId
  }
  return ""
}

function Get-Username($powershellVersion) {
  $username = Get-PublicSettings-From-Config-Json "username"  $powershellVersion
  if ( $username){
    return $username
  }
  return ""
}

function Get-Elasticsearch-URL($powershellVersion) {
  $powershellVersion = Get-PowershellVersion
  $cloudId = Get-CloudId $powershellVersion
  if ( $cloudId -ne ""){
    $cloudHash=$cloudId.split(":")[-1]
    $cloudTokens=[System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($cloudHash))
    $cloudElems=$cloudTokens.split("$")
    $hostPort= $cloudElems[0]
    return "https://$($cloudElems[1]).$(${hostPort})"
  }
  return ""
}

function Get-Kibana-URL ($powershellVersion){
  $cloudId = Get-CloudId $powershellVersion
  if ( $cloudId -ne ""){
     $cloudHash=$cloudId.split(":")[-1]
     $cloudTokens=[System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($cloudHash))
     $cloudElems=$cloudTokens.split("$")
     $hostPort= $cloudElems[0]
    return "https://$($cloudElems[2]).$(${hostPort})"
  }
  return ""
}

function Get-Stack-Version {
  $powershellVersion = Get-PowershellVersion
  $elasticsearchUrl = Get-Elasticsearch-URL $powershellVersion
  $username = Get-Username $powershellVersion
  $password = Get-Password $powershellVersion
  if ( $elasticsearchUrl -ne "" -and $username -ne "" -and $password -ne ""){
    $headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
        $headers.Add("Accept","application/json")
        $pair = "$($username):$($password)"
        $encodedCredentials = [System.Convert]::ToBase64String([System.Text.Encoding]::ASCII.GetBytes($pair))
        $headers.Add('Authorization', "Basic $encodedCredentials")
        $jsonResult = Invoke-WebRequest -Uri "$($elasticsearchUrl)"  -Method 'GET' -Headers $headers -UseBasicParsing
        if ($jsonResult.statuscode -eq '200') {
            $keyValue= ConvertFrom-Json $jsonResult.Content | Select-Object -expand ""
            $stackVersion=$keyValue.version.number
            Write-Log "Found stack version  $stackVersion" "INFO"
            return $stackVersion
             }else {
             Write-Log "Error pinging elastic cluster $elasticsearchUrl" "ERROR"
        return ""
              }
  } else {
    Write-Log "User credentials not found" "ERROR"
  }
  return ""
}

function Get-PublicSettings-From-Config-Json($key, $powershellVersion) {
  Try
  {
    if(!$normalizedJson)
    {
      $azureConfigFile = Get-Azure-Config-Path($powershellVersion)
      $jsonContents = Get-Content $azureConfigFile
      $global:normalizedJson = normalize-json($jsonContents)
    }
    if ( $powershellVersion -ge 3 ) {
      $value = ($normalizedJson | ConvertFrom-Json | Select -expand runtimeSettings | Select -expand handlerSettings | Select -expand publicSettings).$key

    }
    else {
      $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
      $value = $ser.DeserializeObject($normalizedJson).runtimeSettings[0].handlerSettings.publicSettings.$key
    }
    $value
  }
  Catch
  {
    $ErrorMessage = $_.Exception.Message
    $FailedItem = $_.Exception.ItemName
    echo "Failed to read file: $FailedItem. The error message was $ErrorMessage"
    throw "Error in Get-PublicSettings-From-Config-Json. Couldn't parse $azureConfigFile"
  }
}

function Get-ProtectedSettings-From-Config-Json($key, $powershellVersion) {
    Try
    {
        if(!$normalizedJson)
        {
            $azureConfigFile = Get-Azure-Config-Path($powershellVersion)
            $jsonContents = Get-Content $azureConfigFile
            $global:normalizedJson = normalize-json($jsonContents)
        }
        if ( $powershellVersion -ge 3 ) {
            $value = ($normalizedJson | ConvertFrom-Json | Select -expand runtimeSettings | Select -expand handlerSettings).$key

        }
        else {
            $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
            $value = $ser.DeserializeObject($normalizedJson).runtimeSettings[0].handlerSettings.$key
        }
        $value
    }
    Catch
    {
        $ErrorMessage = $_.Exception.Message
        $FailedItem = $_.Exception.ItemName
        echo "Failed to read file: $FailedItem. The error message was $ErrorMessage"
        throw "Error in Get-PublicSettings-From-Config-Json. Couldn't parse $azureConfigFile"
    }
}

function Get-Azure-Logs-Path() {
  try
  {
    $powershellVersion = Get-PowershellVersion
    $handlerFile = "$extensionRoot\\HandlerEnvironment.json"
    if ( $powershellVersion -ge 3 ) {
      $logsFolder = (((Get-Content $handlerFile) | ConvertFrom-Json)[0] | Select -expand handlerEnvironment).logFolder
    }
    else {
      add-type -assembly system.web.extensions
      $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
      $logsFolder = ($ser.DeserializeObject($(Get-Content $handlerFile)))[0].handlerEnvironment.logFolder
    }
    return $logsFolder
  }
  catch
  {
    $ErrorMessage = $_.Exception.Message
    $FailedItem = $_.Exception.ItemName
    Write-Host "Failed to read file: $FailedItem. The error message was $ErrorMessage"
    throw "Error in Get-Azure-Config-Path. Couldn't parse the HandlerEnvironment.json file"
  }
}

function Get-Azure-Config-Path($powershellVersion) {
  Try
  {
    $handlerFile = "$extensionRoot\HandlerEnvironment.json"

    if ( $powershellVersion -ge 3 ) {
      $configFolder = (((Get-Content $handlerFile) | ConvertFrom-Json)[0] | Select -expand handlerEnvironment).configFolder
    }
    else {
      add-type -assembly system.web.extensions
      $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
      $configFolder = ($ser.DeserializeObject($(Get-Content $handlerFile)))[0].handlerEnvironment.configFolder
    }

    # Get the last .settings file
    $configFileName = Get-Lastest-Settings-File($configFolder)
    $azureConfigFile = "$configFolder\$configFileName"
    $configFileIsFolder = (Get-Item $azureConfigFile) -is [System.IO.DirectoryInfo]

    # In case of update, the n.settings file doesn't exists initially in the
    # folder of the new extension. Hence using the n.settings file copied into
    # the C:\Elastic folder during enable
    if ( $configFileIsFolder ) {
      Write-Log "n.settings file doesn't exist in the extension folder. Reading from C:\Elastic." "ERROR"
      $configFolder = "C:\Elastic"
      $configFileName = Get-Lastest-Settings-File($configFolder)
      $azureConfigFile = "$configFolder\$configFileName"
    }
    return $azureConfigFile
  }
  Catch
  {
    $ErrorMessage = $_.Exception.Message
    $FailedItem = $_.Exception.ItemName
    Write-Log "Failed to read file: $FailedItem. The error message was $ErrorMessage" "ERROR"
    throw "Error in Get-Azure-Config-Path. Couldn't parse the HandlerEnvironment.json file"
  }
}

function Get-Azure-Status-Path($powershellVersion) {
  Try
  {
    $handlerFile = "$extensionRoot\\HandlerEnvironment.json"

    if ( $powershellVersion -ge 3 ) {
      $statusFolder = (((Get-Content $handlerFile) | ConvertFrom-Json)[0] | Select -expand handlerEnvironment).statusFolder
    }
    else {
      add-type -assembly system.web.extensions
      $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
      $statusFolder = ($ser.DeserializeObject($(Get-Content $handlerFile)))[0].handlerEnvironment.statusFolder
    }
    return $statusFolder
  }
  Catch
  {
    $ErrorMessage = $_.Exception.Message
    $FailedItem = $_.Exception.ItemName
    Write-Log "Failed to read file: $FailedItem. The error message was $ErrorMessage" "ERROR"
    throw "Error in Get-Azure-Config-Path. Couldn't parse the HandlerEnvironment.json file"
  }
}

function Get-Lastest-Settings-File($configFolder) {
  $configFiles = get-childitem $configFolder -recurse | where {$_.extension -eq ".settings"}

  if($configFiles -is [system.array]) {
    $configFileName = $configFiles[-1].Name
  }
  else {
    $configFileName = $configFiles.Name
  }
  return $configFileName
}

function DownloadFile {
    Param(
        [Parameter(Mandatory=$True)]
        [hashtable]$Params,
        [int]$Retries = 3
    )
    $url = $Params['Uri']
    $outFile = $Params['OutFile']
    [int]$trials = 0
    $webClient = New-Object net.webclient
    do {
        try {
            $trials +=1
            $webClient.DownloadFile($url, $outFile)
            Write-Log "Elastic Agent downloaded" "INFO"
            break
        } catch [System.Net.WebException] {
            Write-Log "Problem downloading $url `tTrial $trials `n` tException:  $_.Exception.Message" "ERROR"
            throw "Problem downloading $url `tTrial $trials `n` tException:  $_.Exception.Message"
        }
    }
    while ($trials -lt $Retries)
}

function Get-Lastest-Status-File($statusFolder) {
  $statusFiles = get-childitem $statusFolder -recurse | where {$_.extension -eq ".status"}

  if($statusFiles -is [system.array]) {
    $statusFileName = $statusFiles[-1].Name
  }
  else {
    $statusFileName = $statusFiles.Name
  }
  return $statusFileName
}

function Write-Status
{
 Param
    (
         [Parameter(Mandatory=$true, Position=0)]
         [string] $name,
         [Parameter(Mandatory=$true, Position=1)]
         [string] $operation,
         [Parameter(Mandatory=$true, Position=2)]
         [string] $mainStatus,
         [Parameter(Mandatory=$true, Position=3)]
         [string] $message,
         [Parameter(Mandatory=$true, Position=4)]
         [string] $subname,
         [Parameter(Mandatory=$true, Position=5)]
         [string] $subStatus,
         [Parameter(Mandatory=$true, Position=6)]
         [string] $subMessage,
         [Parameter(Mandatory=$true, Position=7)]
         [string] $sequenceNumber
    )
  #$sequenceNumber = 1
  $code = 0
  $statusPath = Get-Azure-Status-Path
  if ( $statusPath) {
#    $lastStatusFile = Get-Lastest-Status-File($statusPath)
#    if ($lastStatusFile) {
#        $lastSequence =  $lastStatusFile.Split(".")[0]
#        $sequenceNumber = [int]$lastSequence  + 1
#    }
    $statusFile = $statusPath + "\\" + $sequenceNumber + ".status"
    #transitioning, error, success and warning
    if ($subStatus -eq "error") {
        $code = 1
    }
    $timestampUTC = (Get-Date -Format u).Replace(" ", "T")
    $jsonRequest = [ordered]@{
        version="1.0"
        timestampUTC = "$timestampUTC"
        status= @{
            name = "$name"
            operation = "$operation"
            status = "$mainStatus"
            formattedMessage =@{
                    lang = "en-US"
                    message = "$message"
                       }
            substatus = @(
                @{
                   name = "$subName"
                   status = "$subStatus"
                   code = $code
                   formattedMessage =@{
                        lang = "en-US"
                        message = "$subMessage"
                    }
                }
            )
        }
    }
    if ( $(Get-PowershellVersion) -ge 3) {
      ConvertTo-Json -Compress $jsonRequest -Depth 4 | Out-File -filePath $statusFile
    }
  }
}

function normalize-json($json) {
  $json -Join " "
}

function Get-Agent-Id($fileLocation){
    $text = Get-Content -Path "$fileLocation"
    $regex = '(?ms)(^)agent:(?:.+?)id:\s?(.*?)(?:[\r\n]|$)'
    $text = $text -join "`n"
    $OutputText = [regex]::Matches($text, $regex) |
              foreach {$_.Groups[2].Value -split $regex}
    return $OutputText
}

function Get-Default-Policy($content){
    foreach ($policy in $content) {
        if ($policy.name  -like "Default" -And $policy.active -eq "true") {
        return $policy.id
          }
    }
}

function Get-AnyActive-Policy($content){
    foreach ($policy in $content) {
        if ($policy.active -eq "true") {
        return $policy.id
          }
    }
}

#region encryption

Function Encrypt {
    [CmdletBinding()]
    [OutputType([System.String])]
    param(
        [Parameter(Position=0, Mandatory=$true)][ValidateNotNullOrEmpty()][System.String]
        $ClearText,
        [Parameter(Position=1, Mandatory=$true)][ValidateNotNullOrEmpty()][System.String]
        $CertThumbprint
    )
    $store = new-object System.Security.Cryptography.X509Certificates.X509Store([System.Security.Cryptography.X509Certificates.StoreLocation]::LocalMachine)
    $store.open([System.Security.Cryptography.X509Certificates.OpenFlags]::ReadOnly)
    $cert = $store.Certificates | Where-Object {$_.thumbprint -eq $CertThumbprint}

    $utf8EncrypedByteArray = [System.Text.Encoding]::UTF8.GetBytes($ClearText)
    $content = New-Object Security.Cryptography.Pkcs.ContentInfo -argumentList (,$utf8EncrypedByteArray)
    $env = New-Object Security.Cryptography.Pkcs.EnvelopedCms $content
    $recpient = (New-Object System.Security.Cryptography.Pkcs.CmsRecipient($cert))
    $env.Encrypt($recpient)
    $base64string = [Convert]::ToBase64String($env.Encode())
    Return $base64string
}

function Decrypt
{
    [CmdletBinding()]
    [OutputType([System.String])]
    param(
        [Parameter(Position=0, Mandatory=$true)][ValidateNotNullOrEmpty()][System.String]
        $EncryptedBase64String,
        [Parameter(Position=1, Mandatory=$true)][ValidateNotNullOrEmpty()][System.String]
        $CertThumbprint
    )
    [System.Reflection.Assembly]::LoadWithPartialName("System.Security") | out-null
    $encryptedByteArray = [Convert]::FromBase64String($EncryptedBase64String)
    $envelope =  New-Object System.Security.Cryptography.Pkcs.EnvelopedCms

    # get certificate from local machine store
    $store = new-object System.Security.Cryptography.X509Certificates.X509Store([System.Security.Cryptography.X509Certificates.StoreLocation]::LocalMachine)
    $store.open([System.Security.Cryptography.X509Certificates.OpenFlags]::ReadOnly)
    $cert = $store.Certificates | Where-Object {$_.thumbprint -eq $CertThumbprint}
    if($cert) {
        $envelope.Decode($encryptedByteArray)
        $envelope.Decrypt($cert)
        $decryptedBytes = $envelope.ContentInfo.Content
        $decryptedResult = [System.Text.Encoding]::UTF8.GetString($decryptedBytes)
        Return $decryptedResult
    }
    Return ""
}

function Get-Password($powershellVersion) {
    Try
    {
        $thumbprint = Get-ProtectedSettings-From-Config-Json "protectedSettingsCertThumbprint"  $powershellVersion
        $protectedSettings = Get-ProtectedSettings-From-Config-Json "protectedSettings"  $powershellVersion
        if ( $thumbprint -ne "" -and $protectedSettings -ne "") {
            $jsonKeys = Decrypt $protectedSettings $thumbprint
            if ($jsonKeys) {
                if(!$normalizedJsonKeys)
                {
                    $global:normalizedJsonKeys = normalize-json($jsonKeys)
                }
                if ( $powershellVersion -ge 3 ) {
                    $value = ($normalizedJsonKeys | ConvertFrom-Json).password

                }
                else {
                    $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
                    $value = $ser.DeserializeObject($normalizedJsonKeys).password
                }
                Return $value
            }
        }
    }
    Catch
    {
        $ErrorMessage = $_.Exception.Message
        $FailedItem = $_.Exception.ItemName
        Write-Log "Failed to read file: $FailedItem. The error message was $ErrorMessage" "ERROR"
        throw "Error in Get-PublicSettings-From-Config-Json. Couldn't parse configuration file"
    }
}

function Get-Base64Auth($powershellVersion) {
    Try
    {
        $thumbprint = Get-ProtectedSettings-From-Config-Json "protectedSettingsCertThumbprint"  $powershellVersion
        $protectedSettings = Get-ProtectedSettings-From-Config-Json "protectedSettings"  $powershellVersion
        if ( $thumbprint -ne "" -and $protectedSettings -ne "") {
            $jsonKeys = Decrypt $protectedSettings $thumbprint
            if ($jsonKeys) {
                if(!$normalizedJsonKeys)
                {
                    $global:normalizedJsonKeys = normalize-json($jsonKeys)
                }
                if ( $powershellVersion -ge 3 ) {
                    $value = ($normalizedJsonKeys | ConvertFrom-Json).base64Auth

                }
                else {
                    $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
                    $value = $ser.DeserializeObject($normalizedJsonKeys).base64Auth
                }
                Return $value
            }
        }
    }
    Catch
    {
        $ErrorMessage = $_.Exception.Message
        $FailedItem = $_.Exception.ItemName
        Write-Log "Failed to read file: $FailedItem. The error message was $ErrorMessage" "ERROR"
        throw "Error in Get-ProtectedSettings-From-Config-Json. Couldn't parse configuration file"
    }
}
