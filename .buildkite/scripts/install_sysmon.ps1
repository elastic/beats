$downloadUrl = "https://live.sysinternals.com/Sysmon64.exe"
$tempFolder = "$env:TEMP\SysmonDownload"
$sysmonPath = "$tempFolder\Sysmon64.exe"

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

# Sysmon64.exe can stay locked for a moment after it exits, for example while
# antivirus scans it. The folder is no longer needed, so failing to remove it
# must not fail the job.
$attempts=0
do
{
    $attempts++
    try {
        Remove-Item -Path $tempFolder -Force -Recurse -ErrorAction Stop
        break
    }
    catch {
        if ($attempts -ge 5) {
            Write-Warning "Could not remove ${tempFolder}: $_"
            break
        }
        Start-Sleep -Seconds 2
    }
} while ($true)
