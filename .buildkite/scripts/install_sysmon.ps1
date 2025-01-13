$downloadUrl = "https://download.sysinternals.com/files/Sysmon.zip"
$tempFolder = "$env:TEMP\SysmonDownload"

if (!(Test-Path $tempFolder)) {
    New-Item -ItemType Directory -Path $tempFolder
}

$ProgressPreference = 'SilentlyContinue'
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile "$tempFolder\Sysmon.zip" -UseBasicParsing
} catch {
    $result = $_.Exception.Response.GetResponseStream()
    $reader = New-Object System.IO.StreamReader($result)
    $reader.BaseStream.Position = 0
    $reader.DiscardBufferedData()
    $reader.ReadToEnd()
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
