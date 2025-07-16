param(
    [Parameter(Mandatory=$true)]
    [ValidateSet("Register", "Unregister", "Status")]
    [string]$Action,
    
    [string]$ManifestPath = "sample.man",
    [string]$DllPath = "sample.dll",
    [string]$WevtutilPath = "",
    [switch]$Force = $false
)

$ErrorActionPreference = "Stop"

function Find-Wevtutil {
    param(
        [string]$ProvidedPath = ""
    )

    $foundTool = $null

    if ($ProvidedPath -ne "") {
        if (Test-Path $ProvidedPath -PathType Leaf) {
            $foundTool = $ProvidedPath
            Write-Verbose "Using provided wevtutil path: $foundTool"
        } else {
            Write-Error "Provided wevtutil path not found or is not a file: '$ProvidedPath'"
            return $null
        }
    } else {
        # Try to find wevtutil in PATH first
        $toolFromPath = Get-Command "wevtutil.exe" -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source
        if ($toolFromPath) {
            $foundTool = $toolFromPath
            Write-Host "Found wevtutil.exe in PATH: $foundTool" -ForegroundColor Green
        } else {
            # Try common Windows locations
            $commonPaths = @(
                "${env:SystemRoot}\System32\wevtutil.exe",
                "${env:SystemRoot}\SysWOW64\wevtutil.exe"
            )
            
            foreach ($path in $commonPaths) {
                if (Test-Path $path -PathType Leaf) {
                    $foundTool = $path
                    Write-Host "Found wevtutil.exe: $foundTool" -ForegroundColor Green
                    break
                }
            }
        }
    }

    if (-not $foundTool) {
        Write-Error "wevtutil.exe not found. This tool is required for ETW provider management."
        return $null
    }

    return $foundTool
}

function Test-AdminPrivileges {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Get-ProviderInfo {
    param(
        [string]$ManifestPath
    )
    
    if (-not (Test-Path $ManifestPath -PathType Leaf)) {
        Write-Error "Manifest file not found: $ManifestPath"
        return $null
    }

    try {
        [xml]$manifest = Get-Content $ManifestPath
        $provider = $manifest.instrumentationManifest.instrumentation.events.provider
        
        return @{
            Name = $provider.name
            Guid = $provider.guid
            Symbol = $provider.symbol
            ResourceFileName = $provider.resourceFileName
            MessageFileName = $provider.messageFileName
        }
    } catch {
        Write-Error "Failed to parse manifest file: $($_.Exception.Message)"
        return $null
    }
}

function Register-EtwProvider {
    param(
        [string]$ManifestPath,
        [string]$DllPath,
        [string]$WevtutilExe
    )

    Write-Host "`n--- Registering ETW Provider ---" -ForegroundColor Blue
    
    # Resolve full paths
    $manifestFullPath = (Resolve-Path $ManifestPath).Path
    $dllFullPath = (Resolve-Path $DllPath).Path
    
    Write-Host "Manifest: $manifestFullPath"
    Write-Host "DLL: $dllFullPath"
    
    # Get provider info
    $providerInfo = Get-ProviderInfo -ManifestPath $manifestFullPath
    if (-not $providerInfo) {
        return $false
    }
    
    Write-Host "Provider Name: $($providerInfo.Name)"
    Write-Host "Provider GUID: $($providerInfo.Guid)"

    try {
        # Install the manifest
        Write-Host "`nInstalling manifest..." -ForegroundColor Green
        $installArgs = @("im", $manifestFullPath, "/rf:$dllFullPath", "/mf:$dllFullPath")
        
        Write-Verbose "Running: `"$WevtutilExe`" $($installArgs -join ' ')"
        $result = Start-Process -FilePath $WevtutilExe -ArgumentList $installArgs -Wait -NoNewWindow -PassThru
        
        if ($result.ExitCode -eq 0) {
            Write-Host "Provider registered successfully!" -ForegroundColor Green
            Write-Host "You can now use this provider GUID in your ETW tracing: $($providerInfo.Guid)" -ForegroundColor Yellow
            return $true
        } else {
            Write-Error "Failed to register provider. wevtutil.exe returned exit code: $($result.ExitCode)"
            return $false
        }
    } catch {
        Write-Error "Failed to register provider: $($_.Exception.Message)"
        return $false
    }
}

function Unregister-EtwProvider {
    param(
        [string]$ManifestPath,
        [string]$WevtutilExe
    )

    Write-Host "`n--- Unregistering ETW Provider ---" -ForegroundColor Blue
    
    # Resolve full path
    $manifestFullPath = (Resolve-Path $ManifestPath).Path
    Write-Host "Manifest: $manifestFullPath"
    
    # Get provider info
    $providerInfo = Get-ProviderInfo -ManifestPath $manifestFullPath
    if (-not $providerInfo) {
        return $false
    }
    
    Write-Host "Provider Name: $($providerInfo.Name)"
    Write-Host "Provider GUID: $($providerInfo.Guid)"

    try {
        # Uninstall the manifest
        Write-Host "`nUninstalling manifest..." -ForegroundColor Green
        $uninstallArgs = @("um", $manifestFullPath)
        
        Write-Verbose "Running: `"$WevtutilExe`" $($uninstallArgs -join ' ')"
        $result = Start-Process -FilePath $WevtutilExe -ArgumentList $uninstallArgs -Wait -NoNewWindow -PassThru
        
        if ($result.ExitCode -eq 0) {
            Write-Host "Provider unregistered successfully!" -ForegroundColor Green
            return $true
        } else {
            Write-Error "Failed to unregister provider. wevtutil.exe returned exit code: $($result.ExitCode)"
            return $false
        }
    } catch {
        Write-Error "Failed to unregister provider: $($_.Exception.Message)"
        return $false
    }
}

function Get-EtwProviderStatus {
    param(
        [string]$ManifestPath,
        [string]$WevtutilExe
    )

    Write-Host "`n--- ETW Provider Status ---" -ForegroundColor Blue
    
    # Get provider info from manifest
    $providerInfo = Get-ProviderInfo -ManifestPath $ManifestPath
    if (-not $providerInfo) {
        return $false
    }
    
    Write-Host "Provider Name: $($providerInfo.Name)"
    Write-Host "Provider GUID: $($providerInfo.Guid)"

    try {
        # Check if provider is registered by querying it
        Write-Host "`nChecking provider registration..." -ForegroundColor Green
        
        # Try multiple approaches to check provider status
        $status = $false
        
        # Method 1: Try to query by provider name
        $queryArgs = @("gp", $providerInfo.Name)
        Write-Verbose "Running: `"$WevtutilExe`" $($queryArgs -join ' ')"
        $result = Start-Process -FilePath $WevtutilExe -ArgumentList $queryArgs -Wait -NoNewWindow -PassThru -RedirectStandardOutput "temp_output.txt" -RedirectStandardError "temp_error.txt"
        
        if ($result.ExitCode -eq 0) {
            Write-Host "Provider is registered and active! (Method: Direct name query)" -ForegroundColor Green
            $output = Get-Content "temp_output.txt" -ErrorAction SilentlyContinue
            if ($output) {
                Write-Host "`nProvider Details:" -ForegroundColor Yellow
                $output | ForEach-Object { Write-Host "  $_" }
            }
            $status = $true
        } else {
            # Method 2: Try to query by GUID
            Write-Verbose "Provider name query failed, trying GUID query..."
            $guidQuery = $providerInfo.Guid.Trim('{}')
            $queryGuidArgs = @("gp", $guidQuery)
            Write-Verbose "Running: `"$WevtutilExe`" $($queryGuidArgs -join ' ')"
            $guidResult = Start-Process -FilePath $WevtutilExe -ArgumentList $queryGuidArgs -Wait -NoNewWindow -PassThru -RedirectStandardOutput "temp_output2.txt" -RedirectStandardError "temp_error2.txt"
            
            if ($guidResult.ExitCode -eq 0) {
                Write-Host "Provider is registered! (Method: GUID query)" -ForegroundColor Green
                $output = Get-Content "temp_output2.txt" -ErrorAction SilentlyContinue
                if ($output) {
                    Write-Host "`nProvider Details:" -ForegroundColor Yellow
                    $output | ForEach-Object { Write-Host "  $_" }
                }
                $status = $true
            } else {
                # Method 3: Check if manifest is installed by listing all publishers
                Write-Verbose "GUID query failed, checking if publisher exists in registry..."
                $listArgs = @("ep")
                Write-Verbose "Running: `"$WevtutilExe`" $($listArgs -join ' ')"
                $listResult = Start-Process -FilePath $WevtutilExe -ArgumentList $listArgs -Wait -NoNewWindow -PassThru -RedirectStandardOutput "temp_list.txt" -RedirectStandardError "temp_list_error.txt"
                
                if ($listResult.ExitCode -eq 0) {
                    $publishers = Get-Content "temp_list.txt" -ErrorAction SilentlyContinue
                    if ($publishers -and ($publishers -match [regex]::Escape($providerInfo.Name) -or $publishers -match [regex]::Escape($providerInfo.Guid))) {
                        Write-Host "Provider is registered! (Method: Publisher list search)" -ForegroundColor Green
                        $status = $true
                    } else {
                        Write-Host "Provider is not registered or not found." -ForegroundColor Red
                        $error = Get-Content "temp_error.txt" -ErrorAction SilentlyContinue
                        if ($error) {
                            Write-Host "Error details: $($error -join ' ')" -ForegroundColor Red
                        }
                        $status = $false
                    }
                } else {
                    Write-Host "Provider is not registered or not found." -ForegroundColor Red
                    $error = Get-Content "temp_error.txt" -ErrorAction SilentlyContinue
                    if ($error) {
                        Write-Host "Error details: $($error -join ' ')" -ForegroundColor Red
                    }
                    $status = $false
                }
                
                # Clean up additional temp files
                Remove-Item "temp_list.txt" -ErrorAction SilentlyContinue
                Remove-Item "temp_list_error.txt" -ErrorAction SilentlyContinue
            }
            
            # Clean up GUID query temp files
            Remove-Item "temp_output2.txt" -ErrorAction SilentlyContinue
            Remove-Item "temp_error2.txt" -ErrorAction SilentlyContinue
        }
        
        # Clean up temp files
        Remove-Item "temp_output.txt" -ErrorAction SilentlyContinue
        Remove-Item "temp_error.txt" -ErrorAction SilentlyContinue
        
        return $status
    } catch {
        Write-Error "Failed to check provider status: $($_.Exception.Message)"
        Remove-Item "temp_output.txt" -ErrorAction SilentlyContinue
        Remove-Item "temp_error.txt" -ErrorAction SilentlyContinue
        return $false
    }
}

# Main execution
Write-Host "ETW Provider Management Script" -ForegroundColor Cyan
Write-Host "==============================" -ForegroundColor Cyan

# Check if running as administrator
if (-not (Test-AdminPrivileges)) {
    Write-Error "This script requires administrator privileges. Please run as administrator."
    exit 1
}

# Validate input files
if ($Action -ne "Status") {
    if (-not (Test-Path $ManifestPath -PathType Leaf)) {
        Write-Error "Manifest file not found: $ManifestPath"
        exit 1
    }
    
    if ($Action -eq "Register" -and -not (Test-Path $DllPath -PathType Leaf)) {
        Write-Error "DLL file not found: $DllPath"
        exit 1
    }
}

# Find wevtutil.exe
$wevtutilExe = Find-Wevtutil -ProvidedPath $WevtutilPath
if (-not $wevtutilExe) {
    exit 1
}

# Execute the requested action
$success = $false
try {
    switch ($Action) {
        "Register" {
            $success = Register-EtwProvider -ManifestPath $ManifestPath -DllPath $DllPath -WevtutilExe $wevtutilExe
        }
        "Unregister" {
            $success = Unregister-EtwProvider -ManifestPath $ManifestPath -WevtutilExe $wevtutilExe
        }
        "Status" {
            $success = Get-EtwProviderStatus -ManifestPath $ManifestPath -WevtutilExe $wevtutilExe
        }
    }
} catch {
    Write-Error "An unexpected error occurred: $($_.Exception.Message)"
    exit 1
}

if ($success) {
    Write-Host "`nOperation completed successfully!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "`nOperation failed!" -ForegroundColor Red
    exit 1
}
