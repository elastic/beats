function GetDirectory
{
    $Invocation = (Get-Variable MyInvocation -Scope 1).Value
    Split-Path $Invocation.MyCommand.Path
}

$scriptDir = GetDirectory

$extensionRoot = [System.IO.Path]::GetFullPath("$scriptDir\\..")
$deploy = $extensionRoot + "\settings\deploy\"
$scriptDir = $extensionRoot + "\handler\"

function Add-Zip
{
    param([string]$zipfilename)
    $dir = $scriptDir + $zipfilename
    $compress = @{
        LiteralPath= "$dir\HandlerManifest.json", "$dir\scripts"
        CompressionLevel = "Fastest"
        DestinationPath = "$deploy\$zipfilename.zip"
    }
    Compress-Archive @compress  -Force
}

Add-Zip "windows"
Add-Zip "linux"

