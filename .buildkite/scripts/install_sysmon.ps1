$downloadUrl = "https://download.sysinternals.com/files/Sysmon.zip"
$tempFolder = "$env:TEMP\SysmonDownload"

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

try {
    $result = Invoke-WebRequest -Uri $downloadUrl -OutFile "$tempFolder\Sysmon.zip" -UseBasicParsing
}
catch {
    $resp = ParseErrorForResponseBody($_)
    Write-Host "$resp"
    exit 1
}

Write-Host "Sysmon.zip downloaded successfully."

Expand-Archive -Path "$tempFolder\Sysmon.zip" -DestinationPath $tempFolder

$sysmonPath = Get-ChildItem -Path "$tempFolder" -Filter "Sysmon64.exe" | Select-Object -ExpandProperty FullName

if ($sysmonPath) {
    Start-Process -FilePath $sysmonPath -ArgumentList "-m" -Wait

    Write-Host "Sysmon event manifest installation completed."
} else {
    Write-Host "Sysmon executable not found in the downloaded archive."
}

# Clean up the downloaded file
Remove-Item -Path "$tempFolder\Sysmon.zip"
Remove-Item -Path $tempFolder -Force -Recurse
