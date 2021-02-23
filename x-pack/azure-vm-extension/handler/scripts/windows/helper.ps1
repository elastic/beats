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
  $cloud_id = Get-PublicSettings-From-Config-Json "cloud_id"  $powershellVersion
  if ( $cloud_id){
    return $cloud_id
  } else {
    echo "Cloud ID not found."
  }
  return ""
}
function Get-Username($powershellVersion) {
  $cloud_id = Get-PublicSettings-From-Config-Json "username"  $powershellVersion
  if ( $cloud_id){
    return $cloud_id
  } else {
    echo "Cloud ID not found."
  }
  return ""
}
function Get-Password($powershellVersion) {
  $cloud_id = Get-PublicSettings-From-Config-Json "password"  $powershellVersion
  if ( $cloud_id){
    return $cloud_id
  } else {
    echo "Cloud ID not found."
  }
  return ""
}

function Get-Elasticsearch-URL($powershellVersion) {
  $powershellVersion = Get-PowershellVersion
  $cloud_id = Get-CloudId $powershellVersion
  if ( $cloud_id -ne ""){
    $cloud_hash=$cloud_id.split(":")[-1]
    $cloud_tokens=[System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($cloud_hash))
    cloud_elem=$cloud_tokens.split("$")
    $host_port= $cloud_elem[0]
    return "https://$($cloud_elem[1]).$(${host_port})"
  } else {
    echo "Cloud ID not found."
  }
  return ""
}
function Get-Kibana-URL ($powershellVersion){
  $cloud_id = Get-CloudId $powershellVersion
  if ( $cloud_id -ne ""){
    $cloud_hash=$cloud_id.split(":")[-1]
    $cloud_tokens=[System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($cloud_hash))
    cloud_elem=$cloud_tokens.split("$")
    $host_port= $cloud_elem[0]
    return "https://$($cloud_elem[2]).$(${host_port})"
  } else {
    echo "Cloud ID not found."
  }
  return ""
}



function Get-Stack-Version {
  $powershellVersion = Get-PowershellVersion
  $elasticsearch_url = Get-Elasticsearch-URL $powershellVersion
  $username = Get-Username $powershellVersion
  $password = Get-Password $powershellVersion
  if ( $elasticsearch_url -ne "" -and $username -ne "" -and $password -ne ""){
    $headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
        $headers.Add("Accept","Application/Json")
        $pair = "$($username):$($password)"
        $encodedCredentials = [System.Convert]::ToBase64String([System.Text.Encoding]::ASCII.GetBytes($pair))
        $headers.Add('Authorization', "Basic $encodedCredentials")
        $jsonResult = Invoke-WebRequest -Uri "$($elasticsearch_url)"  -Method 'GET' -Headers $headers -UseBasicParsing
        if ($jsonResult.statuscode -eq '200') {
            $keyValue= ConvertFrom-Json $jsonResult.Content | Select-Object -expand "item"
            $stack_version=$keyValue.version.number
            Write-Log "Found stack version  $stack_version" "INFO"
            return $stack_version
             }else {
             Write-Log "Error pinging elastic cluster $elasticsearch_url" "ERROR"
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
    if(!$normalized_json)
    {
      $azure_config_file = Get-Azure-Config-Path($powershellVersion)
      $json_contents = Get-Content $azure_config_file
      $global:normalized_json = normalize_json($json_contents)
    }
    if ( $powershellVersion -ge 3 ) {
      $value = ($normalized_json | ConvertFrom-Json | Select -expand runtimeSettings | Select -expand handlerSettings | Select -expand publicSettings).$key
    }
    else {
      $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
      $value = $ser.DeserializeObject($normalized_json).runtimeSettings[0].handlerSettings.publicSettings.$key
    }
    $value
  }
  Catch
  {
    $ErrorMessage = $_.Exception.Message
    $FailedItem = $_.Exception.ItemName
    echo "Failed to read file: $FailedItem. The error message was $ErrorMessage"
    throw "Error in Get-PublicSettings-From-Config-Json. Couldn't parse $azure_config_file"
  }
}

function Get-Azure-Logs-Path($powershellVersion) {
  try
  {
    $handler_file = "$extensionRoot\\HandlerEnvironment.json"

    if ( $powershellVersion -ge 3 ) {
      $config_folder = (((Get-Content $handler_file) | ConvertFrom-Json)[0] | Select -expand handlerEnvironment).logsFolder
    }
    else {
      add-type -assembly system.web.extensions
      $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
      $config_folder = ($ser.DeserializeObject($(Get-Content $handler_file)))[0].handlerEnvironment.logsFolder
    }
    return $config_file
  }
  catch
  {
    $ErrorMessage = $_.Exception.Message
    $FailedItem = $_.Exception.ItemName
    echo "Failed to read file: $FailedItem. The error message was $ErrorMessage"
    throw "Error in Get-Azure-Config-Path. Couldn't parse the HandlerEnvironment.json file"
  }
}


function Get-Azure-Config-Path($powershellVersion) {
  Try
  {
    $handler_file = "$extensionRoot\\HandlerEnvironment.json"

    if ( $powershellVersion -ge 3 ) {
      $config_folder = (((Get-Content $handler_file) | ConvertFrom-Json)[0] | Select -expand handlerEnvironment).configFolder
    }
    else {
      add-type -assembly system.web.extensions
      $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
      $config_folder = ($ser.DeserializeObject($(Get-Content $handler_file)))[0].handlerEnvironment.configFolder
    }

    # Get the last .settings file
    $config_file_name = Get-Lastest-Settings-File($config_folder)

    $azure_config_file = "$config_folder\$config_file_name"
    $config_file_is_a_folder = (Get-Item $azure_config_file) -is [System.IO.DirectoryInfo]

    # In case of update, the n.settings file doesn't exists initially in the
    # folder of the new extension. Hence using the n.settings file copied into
    # the C:\Chef folder during enable
    if ( $config_file_is_a_folder ) {
      Write-Host "n.settings file doesn't exist in the extension folder. Reading from C:\Elastic."
      $config_folder = "C:\Elastic"
      $config_file_name = Get-Lastest-Settings-File($config_folder)
      $azure_config_file = "$config_folder\$config_file_name"
    }
    return $azure_config_file
  }
  Catch
  {
    $ErrorMessage = $_.Exception.Message
    $FailedItem = $_.Exception.ItemName
    echo "Failed to read file: $FailedItem. The error message was $ErrorMessage"
    throw "Error in Get-Azure-Config-Path. Couldn't parse the HandlerEnvironment.json file"
  }
}

function Get-Lastest-Settings-File($config_folder) {
  $config_files = get-childitem $config_folder -recurse | where {$_.extension -eq ".settings"}

  if($config_files -is [system.array]) {
    $config_file_name = $config_files[-1].Name
  }
  else {
    $config_file_name = $config_files.Name
  }
  return $config_file_name
}

function DownloadFile {
Param(
        [Parameter(Mandatory=$True)]
        [hashtable]$Params,
        [int]$Retries = 3
    )
    $url = $Params['Uri']
    $out_file = $Params['OutFile']
[int]$trials = 0
echo $url
$webClient = New-Object net.webclient
do {
    try {
        $trials +=1
        $webClient.DownloadFile($url, $out_file)
         Write-Log "Elastic Agent downloaded" "INFO"
        break
    } catch [System.Net.WebException] {
    Write-Log "Problem downloading $url `tTrial $trials `n` tException:  $_.Exception.Message" "ERROR"
    throw "Problem downloading $url `tTrial $trials `n` tException:  $_.Exception.Message"
    }
}
while ($trials -lt $Retries)
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
    # Decrypts cipher text using the private key
    # Assumes the certificate is in the LocalMachine\My (Personal) Store
    $Cert = Get-ChildItem cert:\LocalMachine\My | where { $_.Thumbprint -eq $CertThumbprint }
    if($Cert) {
        $EncryptedByteArray = [Convert]::FromBase64String($EncryptedBase64String)
        $ClearText = [System.Text.Encoding]::UTF8.GetString($Cert.PrivateKey.Decrypt($EncryptedByteArray,$true))
    }
    Else {Write-Error "Certificate with thumbprint: $CertThumbprint not found!"}

    Return $ClearText
}

Function Encrypt {
    [CmdletBinding()]
    [OutputType([System.String])]
    param(
        [Parameter(Position=0, Mandatory=$true)][ValidateNotNullOrEmpty()][System.String]
        $ClearText,
        [Parameter(Position=1, Mandatory=$true)][ValidateNotNullOrEmpty()][ValidateScript({Test-Path $_ -PathType Leaf})][System.String]
        $PublicCertFilePath
    )
    # Encrypts a string with a public key
    $PublicCert = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2($PublicCertFilePath)
    $ByteArray = [System.Text.Encoding]::UTF8.GetBytes($ClearText)
    $EncryptedByteArray = $PublicCert.PublicKey.Key.Encrypt($ByteArray,$true)
    $Base64String = [Convert]::ToBase64String($EncryptedByteArray)

    Return $Base64String
}
