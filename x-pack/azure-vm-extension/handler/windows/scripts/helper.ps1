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
  return $powershellVersion
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

function Get-Kibana-URL ($powershellVersion) {
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
  if (-Not $elasticsearchUrl) {
      throw "Elasticsearch URL could not be found"
  }
  $password = Get-Password $powershellVersion
  $base64Auth = Get-Base64Auth $powershellVersion
  if (-Not $password -And -Not $base64Auth) {
      throw "Password  or base64auto key could not be found"
  }
  $headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
  $headers.Add("Accept","application/json")
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
  return ""
}

function Get-PublicSettings-From-Config-Json($key, $powershellVersion) {
    Try
    {
      $azureConfigFile = Get-Azure-Latest-Config-File($powershellVersion)
      $jsonContents = Get-Content $azureConfigFile
      $normalizedJson = normalize-json($jsonContents)
        if ( $powershellVersion -ge 3 ) {
            $keyVal = ($normalizedJson | ConvertFrom-Json | Select -expand runtimeSettings | Select -expand handlerSettings | Select -expand publicSettings).$key
        }
        else {
            $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
            $keyVal = $ser.DeserializeObject($normalizedJson).runtimeSettings[0].handlerSettings.publicSettings.$key
        }
        return $keyVal
    }
    Catch
    {
        $ErrorMessage = $_.Exception.Message
        $FailedItem = $_.Exception.ItemName
        Write-Log "Failed to read file: $FailedItem. The error message was $ErrorMessage" "ERROR"
        throw "Error in Get-PublicSettings-From-Config-Json. Couldn't parse $azureConfigFile"
    }
}

function Get-ProtectedSettings-From-Config-Json($key, $powershellVersion) {
    Try
    {
        $azureConfigFile = Get-Azure-Latest-Config-File($powershellVersion)
        $jsonContents = Get-Content $azureConfigFile
        $normalizedJson = normalize-json($jsonContents)
        if ( $powershellVersion -ge 3 ) {
            $keyVal = ($normalizedJson | ConvertFrom-Json | Select -expand runtimeSettings | Select -expand handlerSettings).$key
        }
        else {
            $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
            $keyVal = $ser.DeserializeObject($normalizedJson).runtimeSettings[0].handlerSettings.$key
        }
        return $keyVal
    }
    Catch
    {
        $ErrorMessage = $_.Exception.Message
        $FailedItem = $_.Exception.ItemName
        Write-Log "Failed to read file: $FailedItem. The error message was $ErrorMessage" "ERROR"
        throw "Error in Get-ProtectedSettings-From-Config-Json. Couldn't parse $azureConfigFile"
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
    throw "Error in Get-Azure-Logs-Path. Couldn't parse the HandlerEnvironment.json file"
  }
}

function Get-Azure-Latest-Config-File($powershellVersion) {
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
    $configFileName = Get-Latest-Settings-File($configFolder)
    $azureConfigFile = "$configFolder\$configFileName"
    Write-Log "The latest configuration file is $azureConfigFile" "INFO"
    $configFileIsFolder = (Get-Item $azureConfigFile) -is [System.IO.DirectoryInfo]

    # In case of update, the n.settings file doesn't exists initially in the
    # folder of the new extension. Hence using the n.settings file copied into
    # the C:\Elastic folder during enable
    if ( $configFileIsFolder ) {
        Write-Log "n.settings file doesn't exist in the extension folder." "ERROR"
        throw "Error in Get-Azure-Latest-Config-File. Missing settings file"
    }
    return $azureConfigFile
  }
  Catch
  {
    $ErrorMessage = $_.Exception.Message
    $FailedItem = $_.Exception.ItemName
    Write-Log "Failed to read file: $FailedItem. The error message was $ErrorMessage" "ERROR"
    throw "Error in Get-Azure-Latest-Config-File. Couldn't parse the HandlerEnvironment.json file"
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
    throw "Error in Get-Azure-Status-Path. Couldn't parse the HandlerEnvironment.json file"
  }
}

function Get-Latest-Settings-File($configFolder) {
  $configFiles = get-childitem $configFolder -recurse | where {$_.extension -eq ".settings"}

  if($configFiles -is [system.array]) {
    $configFileName = $configFiles[-1].Name
  }
  else {
    $configFileName = $configFiles.Name
  }
  return $configFileName
}

function Get-Sequence() {
    $settingsSequence = "0"
    $powershellVersion = Get-PowershellVersion
    $azureConfigFile = Get-Azure-Latest-Config-File($powershellVersion)
    if ($azureConfigFile) {
        $outputFile = Split-Path $azureConfigFile -leaf
        $items = $outputFile.split(".")
        $settingsSequence = $items[0]
    }
    return $settingsSequence
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

function Get-Latest-Status-File($statusFolder) {
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
         [string] $mainMessage,
         [Parameter(Mandatory=$true, Position=4)]
         [string] $subname,
         [Parameter(Mandatory=$true, Position=5)]
         [string] $subStatus,
         [Parameter(Mandatory=$true, Position=6)]
         [string] $subMessage
    )
  $sequenceNumber = Get-Sequence
  $code = 0
  $statusPath = Get-Azure-Status-Path
  if ( $statusPath) {
    $statusFile = $statusPath + "\\" + $sequenceNumber + ".status"
    #transitioning, error, success and warning
    if ($subStatus -eq "error") {
        $code = 1
    }
    $timestampUTC = (Get-Date -Format u).Replace(" ", "T")
    $jsonRequest = @(
      @{
        version="1.0"
        timestampUTC = "$timestampUTC"
        status= @{
            name = "$name"
            operation = "$operation"
            status = "$mainStatus"
            formattedMessage =@{
                lang = "en-US"
                message = "$mainMessage"
            }
            substatus =  @(
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
    )
    if ( $(Get-PowershellVersion) -ge 3) {
      ConvertTo-Json -Compress $jsonRequest -Depth 6 | Out-File -filePath $statusFile
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
                $normalizedJsonKeys = normalize-json($jsonKeys)
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
        throw "Error in Get-ProtectedSettings-From-Config-Json. Couldn't parse configuration file"
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
                $normalizedJsonKeys = normalize-json($jsonKeys)
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


#Env

function Is-New-Config {
    $currentSequence = [Environment]::GetEnvironmentVariable('ELASTICAGENTSEQUENCE', 'Machine')
    $newSequence = [Environment]::GetEnvironmentVariable("ConfigSequenceNumber")
    $isUpdate = [Environment]::GetEnvironmentVariable("ELASTICAGENTUPDATE")
    Write-Log "Current sequence is $currentSequence and new sequence is $newSequence" "INFO"
    if (!$newSequence) {
        return $false
     }
    if ($isUpdate -eq "1") {
        Write-Log "Part of update" "INFO"
        return $false
    }
    if (!$newSequence) {
        return $false
    }
    if ($currentSequence -eq $newSequence ) {
        return $false
    }
    return $true
}

function Set-SequenceEnvVariables
{
    $newSequence = [Environment]::GetEnvironmentVariable("ConfigSequenceNumber")
    if (!$newSequence) {
        $newSequence = Get-Sequence
    }
    [Environment]::SetEnvironmentVariable("ELASTICAGENTSEQUENCE", $newSequence, "Machine")
    [Environment]::SetEnvironmentVariable("ELASTICAGENTUPDATE", 0, "Machine")
}

function Set-UpdateEnvVariables
{
    [Environment]::SetEnvironmentVariable("ELASTICAGENTUPDATE", 1, "Machine")
}

function Get-Prev-Settings-File($configFolder) {
    $configFiles = get-childitem $configFolder -recurse | where {$_.extension -eq ".settings"}
    if($configFiles -is [system.array]) {
        $configFileName = $configFiles[-2].Name
    }
    else {
        $configFileName = $configFiles.Name
    }
    return $configFileName
}

function Get-Azure-Prev-Config-File($powershellVersion) {
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
        $configFileName = Get-Prev-Settings-File($configFolder)
        $azureConfigFile = "$configFolder\$configFileName"
        Write-Log "The previous file is $azureConfigFile" "INFO"
        $configFileIsFolder = (Get-Item $azureConfigFile) -is [System.IO.DirectoryInfo]

        # In case of update, the n.settings file doesn't exists initially in the
        # folder of the new extension. Hence using the n.settings file copied into
        # the C:\Elastic folder during enable
        if ( $configFileIsFolder ) {
            Write-Log "n.settings file doesn't exist in the extension folder." "ERROR"
            throw "Error in Get-Azure-Prev-Config-File. Missing settings file"
        }
        return $azureConfigFile
    }
    Catch
    {
        $ErrorMessage = $_.Exception.Message
        $FailedItem = $_.Exception.ItemName
        Write-Log "Failed to read file: $FailedItem. The error message was $ErrorMessage" "ERROR"
        throw "Error in Get-Azure-Prev-Config-File. Couldn't parse the HandlerEnvironment.json file"
    }
}


function Get-Prev-ProtectedSettings-From-Config-Json($key, $powershellVersion) {
    Try
    {
        $azureConfigFile = Get-Azure-Prev-Config-File($powershellVersion)
        $jsonContents = Get-Content $azureConfigFile
        $normalizedJson = normalize-json($jsonContents)
        if ( $powershellVersion -ge 3 ) {
            $keyVal = ($normalizedJson | ConvertFrom-Json | Select -expand runtimeSettings | Select -expand handlerSettings).$key
        }
        else {
            $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
            $keyVal = $ser.DeserializeObject($normalizedJson).runtimeSettings[0].handlerSettings.$key
        }
        return $keyVal
    }
    Catch
    {
        $ErrorMessage = $_.Exception.Message
        $FailedItem = $_.Exception.ItemName
        Write-Log "Failed to read file: $FailedItem. The error message was $ErrorMessage" "ERROR"
        throw "Error in Get-Prev-ProtectedSettings-From-Config-Json. Couldn't parse $azureConfigFile"
    }
}

function Get-Prev-Password($powershellVersion) {
    Try
    {
        $thumbprint = Get-Prev-ProtectedSettings-From-Config-Json "protectedSettingsCertThumbprint"  $powershellVersion
        $protectedSettings = Get-Prev-ProtectedSettings-From-Config-Json "protectedSettings"  $powershellVersion
        if ( $thumbprint -ne "" -and $protectedSettings -ne "") {
            $jsonKeys = Decrypt $protectedSettings $thumbprint
            if ($jsonKeys) {
                $normalizedJsonKeys = normalize-json($jsonKeys)
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
        throw "Error in Get-Prev-ProtectedSettings-From-Config-Json. Couldn't parse configuration file"
    }
}

function Get-Prev-Base64Auth($powershellVersion) {
    Try
    {
        $thumbprint = Get-Prev-ProtectedSettings-From-Config-Json "protectedSettingsCertThumbprint"  $powershellVersion
        $protectedSettings = Get-Prev-ProtectedSettings-From-Config-Json "protectedSettings"  $powershellVersion
        if ( $thumbprint -ne "" -and $protectedSettings -ne "") {
            $jsonKeys = Decrypt $protectedSettings $thumbprint
            if ($jsonKeys) {
                $normalizedJsonKeys = normalize-json($jsonKeys)
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
        throw "Error in Get-Prev-ProtectedSettings-From-Config-Json. Couldn't parse configuration file"
    }
}

function Get-Prev-PublicSettings-From-Config-Json($key, $powershellVersion) {
    Try
    {
        $azureConfigFile = Get-Azure-Prev-Config-File($powershellVersion)
        $jsonContents = Get-Content $azureConfigFile
        $normalizedJson = normalize-json($jsonContents)
        if ( $powershellVersion -ge 3 ) {
            $keyVal = ($normalizedJson | ConvertFrom-Json | Select -expand runtimeSettings | Select -expand handlerSettings | Select -expand publicSettings).$key
        }
        else {
            $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
            $keyVal = $ser.DeserializeObject($normalizedJson).runtimeSettings[0].handlerSettings.publicSettings.$key
        }
        return $keyVal
    }
    Catch
    {
        $ErrorMessage = $_.Exception.Message
        $FailedItem = $_.Exception.ItemName
        echo "Failed to read file: $FailedItem. The error message was $ErrorMessage"
        throw "Error in Get-Prev-PublicSettings-From-Config-Json. Couldn't parse $azureConfigFile"
    }
}

function Get-Prev-CloudId($powershellVersion) {
    $cloudId = Get-Prev-PublicSettings-From-Config-Json "cloudId"  $powershellVersion
    if ( $cloudId){
        return $cloudId
    }
    return ""
}

function Get-Prev-Username($powershellVersion) {
    $username = Get-Prev-PublicSettings-From-Config-Json "username"  $powershellVersion
    if ( $username){
        return $username
    }
    return ""
}

function Get-Prev-Kibana-URL($powershellVersion) {
    $cloudId = Get-Prev-CloudId $powershellVersion
    if ( $cloudId -ne ""){
        $cloudHash=$cloudId.split(":")[-1]
        $cloudTokens=[System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($cloudHash))
        $cloudElems=$cloudTokens.split("$")
        $hostPort= $cloudElems[0]
        return "https://$($cloudElems[2]).$(${hostPort})"
    }
    return ""
}
