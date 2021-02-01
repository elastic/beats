log()
{
    Write-Host \[$(date +%d%m%Y-%H:%M:%S)\] "$1"
    echo \[$(date +%d%m%Y-%H:%M:%S)\] "$1" >> C:\logs\es-agent-install.log
}

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

function Request {
    Param(
        [Parameter(Mandatory=$True)]
        [hashtable]$Params,
        [int]$Retries = 1,
        [int]$SecondsDelay = 2
    )

    $method = $Params['Method']
    $url = $Params['Uri']
    $out_file = $Params['OutFile']
    $cmd = { Write-Host "$method  $url -OutFile $out_file" -NoNewline; Invoke-WebRequest @Params }

    $retryCount = 0
    $completed = $false
    $response = $null

    while (-not $completed) {
        try {
            $response = Invoke-Command $cmd -ArgumentList $Params
            if ($response.StatusCode -ne 200) {
                throw "Expecting response code 200, was: $($response.StatusCode)"
            }
            $completed = $true
        } catch {
            New-Item -ItemType Directory -Force -Path C:\logs\
            "$(Get-Date -Format G): Request to $url failed. $_" | Out-File -FilePath 'C:\logs\vm.log' -Encoding utf8 -Append
            if ($retrycount -ge $Retries) {
                Write-Warning "Request to $url failed the maximum number of $retryCount times."
                throw
            } else {
                Write-Warning "Request to $url failed. Retrying in $SecondsDelay seconds."
                Start-Sleep $SecondsDelay
                $retrycount++
            }
        }
    }
    Write-Host "OK ($($response.StatusCode))"
    return $response
}

function es_agent_install {
 $OS_SUFFIX="-windows-x86_64"
 $STACK_VERSION="7.10.2"
 $PACKAGE="elastic-agent-${STACK_VERSION}${OS_SUFFIX}.zip"
 $ALGORITHM="512"
 $SHASUM="$PACKAGE.sha$ALGORITHM"
 $DOWNLOAD_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}"
 $SHASUM_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}.sha512"
 echo $DOWNLOAD_URL

 $req = Request -Params @{ 'Method'='GET';'Uri'="$DOWNLOAD_URL";'OutFile'='C:\tests\file'}
 echo $req
}


es_agent_install
