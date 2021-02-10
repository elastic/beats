$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)

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


function es_agent_install {
    #For testing only
    $STACK_VERSION=$env:STACK_VERSION
    $CLOUD_ID=$env:CLOUD_ID
    $USERNAME=$env:USERNAME
    $PASSWORD=$env:PASSWORD
    #end
    $OS_SUFFIX="-windows-x86_64"
    $INSTALL= "elastic-agent-${STACK_VERSION}${OS_SUFFIX}"
    $PACKAGE="${INSTALL}.zip"
    $ALGORITHM="512"
    $SHASUM="$PACKAGE.sha$ALGORITHM"
    $DOWNLOAD_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}"
    $SHASUM_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}.sha512"
    $SAVEDFILE="$env:temp\" + $PACKAGE
    $INSTALL_LOCATION="C:\Program Files"
    try {
        Write-Log "Started" "INFO"
        DownloadFile -Params @{'Uri'="$DOWNLOAD_URL";'OutFile'="$SAVEDFILE"}
        Write-Log "Unzip elastic agent archive" "INFO"
        Expand-Archive -LiteralPath $SAVEDFILE -DestinationPath $INSTALL_LOCATION
        Write-Log "Elastic agent unzipped location $INSTALL_LOCATION" "INFO"
        Write-Log "Rename folder ..."
        Rename-Item -Path "$INSTALL_LOCATION\$INSTALL" -NewName "Elastic-Agent"
        Write-Log "Folder $INSTALL renamed to 'Agent'"
    }
    catch {
      Write-Log "An error occurred:" "ERROR"
      Write-Log $_ "ERROR"
      Write-Log $_.ScriptStackTrace "ERROR"
    }

}


es_agent_install
