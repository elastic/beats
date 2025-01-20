$downloadUrl = "https://live.sysinternals.com/Sysmon64.exe"
$tempFolder = "$env:TEMP\SysmonDownload"
$sysmonPath = "$tempFolder\Sysmon64.exe"

function Retry()
{
    param(
        [Parameter(Mandatory=$true)][Action]$action,
        [Parameter(Mandatory=$false)][int]$maxAttempts = 3
    )

    $attempts=1    
    $ErrorActionPreferenceToRestore = $ErrorActionPreference
    $ErrorActionPreference = "Stop"

    do
    {
        try
        {
            $action.Invoke();
            break;
        }
        catch [Exception]
        {
            Write-Host $_.Exception.Message
        }

        # exponential backoff delay
        $attempts++
        if ($attempts -le $maxAttempts) {
            Write-Host("Action failed. Waiting " + $retryDelaySeconds + " seconds before attempt " + $attempts + " of " + $maxAttempts + ".")
            Start-Sleep 5
        }
        else {
            $ErrorActionPreference = $ErrorActionPreferenceToRestore
            Write-Error $_.Exception.Message
        }
    } while ($attempts -le $maxAttempts)
    $ErrorActionPreference = $ErrorActionPreferenceToRestore
}

if (!(Test-Path $tempFolder)) {
    New-Item -ItemType Directory -Path $tempFolder
}

$ProgressPreference = 'SilentlyContinue'
function ParseErrorForResponseBody($Error) {
    if ($PSVersionTable.PSVersion.Major -lt 6) {
        if ($Error.Exception.Response) {  
            $Reader = New-Object System.IO.StreamReader($Error.Exception.Response.GetResponseStream())
            $Reader.BaseStream.Position = 0
            $Reader.DiscardBufferedData()
            $ResponseBody = $Reader.ReadToEnd()
            if ($ResponseBody.StartsWith('{')) {
                $ResponseBody = $ResponseBody | ConvertFrom-Json
            }
            return $ResponseBody
        }
    }
    else {
        return $Error.ErrorDetails.Message
    }
}

$attempts=1
do
{
    $attempts++
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $result = Invoke-WebRequest -Uri $downloadUrl -OutFile $sysmonPath -UseBasicParsing
        break
    }
    catch {
        $resp = ParseErrorForResponseBody($_)
        Write-Host "$resp"
        if ($attempts -gt 5) {
            exit 1
        }
    }
} while ($attempts -le 5)

Write-Host "Sysmon64.exe downloaded successfully."

if ($sysmonPath) {
    Start-Process -FilePath $sysmonPath -ArgumentList "-m" -Wait

    Write-Host "Sysmon event manifest installation completed."
} else {
    Write-Host "Sysmon executable not found in the downloaded archive."
}

Remove-Item -Path $tempFolder -Force -Recurse
