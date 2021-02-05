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
