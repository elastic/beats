$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory log.ps1)
$ScriptDirectory = Split-Path $MyInvocation.MyCommand.Path
. (Join-Path $ScriptDirectory helper.ps1)


function Write-Status1
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
         [string] $message,
         [Parameter(Mandatory=$true, Position=4)]
         [string] $subname,
         [Parameter(Mandatory=$true, Position=5)]
         [string] $subStatus,
         [Parameter(Mandatory=$true, Position=6)]
         [string] $subMessage
    )
  $sequenceNumber = 1
  $code = 0
  $statusPath = Get-Azure-Status-Path
  if ( $statusPath) {
    $lastStatusFile = Get-Lastest-Status-File($statusPath)
    if ($lastStatusFile) {
        $lastSequence =  $lastStatusFile.Split(".")[0]
        $sequenceNumber = [int]$lastSequence  + 1
    }
    $statusFile = $statusPath + "\\" + $sequenceNumber + ".status"
    #transitioning, error, success and warning
    if ($subStatus -eq "error") {
        $code = 1
    }
    $timestampUTC = (Get-Date -Format u).Replace(" ", "T")
    $jsonRequest = [ordered]@{
        version="1.0"
        timestampUTC = "$timestampUTC"
        status= @{
            name = "$name"
            operation = "$operation"
            status = "$mainStatus"
            formattedMessage =@{
                    lang = "en-US"
                    message = "$message"
                       }
            substatus = @(
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
    if ( $(Get-PowershellVersion) -ge 3) {
      ConvertTo-Json -Compress $jsonRequest -Depth 4 | Out-File -filePath $statusFile
    }
  }
}


$name= "install_elastic_agent"
$operation = "enroll"
$mainStatus = "transitioning"
$message= "messgae2 "
$subStatus= "error"
$subMessage = "submessage"
$subName = "sds"
Write-Status1 "$name" "$operation" "$mainStatus" "$message" "$subName" "$subStatus" "$subMessage"
