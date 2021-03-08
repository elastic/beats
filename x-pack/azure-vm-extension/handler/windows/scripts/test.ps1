$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

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
                $base64Auth = ($normalizedJsonKeys | ConvertFrom-Json).base64Auth

            }
            else {
                $ser = New-Object System.Web.Script.Serialization.JavaScriptSerializer
                $value = $ser.DeserializeObject($normalizedJsonKeys).password
                $base64Auth = $ser.DeserializeObject($normalizedJsonKeys).base64Auth
            }
            $value
        }
    }
    }
    Catch
    {
        $ErrorMessage = $_.Exception.Message
        $FailedItem = $_.Exception.ItemName
        echo "Failed to read file: $FailedItem. The error message was $ErrorMessage"
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
                $value
            }
        }
        return ""
    }
    Catch
    {
        $ErrorMessage = $_.Exception.Message
        $FailedItem = $_.Exception.ItemName
        echo "Failed to read file: $FailedItem. The error message was $ErrorMessage"
        throw "Error in Get-ProtectedSettings-From-Config-Json. Couldn't parse configuration file"
    }
}

