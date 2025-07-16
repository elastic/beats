param(
    [string]$ManifestFile = "sample.man",
    [string]$OutputPath = ".",
    [string]$McExePath = "",
    [string]$RcExePath = "",
    [string]$LinkExePath = "",
    [switch]$Force = $false
)

$ErrorActionPreference = "Stop"

function Find-Tool {
    param(
        [string]$ToolName,
        [string]$ProvidedPath = "",
        [string]$EnvVarName = ""
    )

    $foundTool = $null

    if ($ProvidedPath -ne "") {
        if (Test-Path $ProvidedPath -PathType Leaf) {
            $foundTool = $ProvidedPath
            Write-Verbose "Using provided $ToolName path: $foundTool"
        } else {
            Write-Error "Provided $ToolName path not found or is not a file: '$ProvidedPath'"
            return $null
        }
    } elseif ($EnvVarName -ne "" -and (Get-Item Env:$EnvVarName -ErrorAction SilentlyContinue)) {
        $envValue = Get-Item Env:$EnvVarName | Select-Object -ExpandProperty Value
        if (Test-Path $envValue -PathType Leaf) {
            $foundTool = $envValue
            Write-Verbose "Using $ToolName from environment variable '$EnvVarName': $foundTool"
        } else {
            Write-Warning "$EnvVarName environment variable points to a non-existent file: '$envValue'. Searching common locations."
        }
    }

    if ($foundTool -eq $null) {
        Write-Verbose "Searching for $ToolName in PATH and common SDK/VS locations..."

        $toolFromPath = Get-Command $ToolName -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source
        if ($toolFromPath) {
            $foundTool = $toolFromPath
            Write-Host "Found $ToolName in PATH: $foundTool" -ForegroundColor Green
        } else {
            # Detect system architecture
            $architecture = $env:PROCESSOR_ARCHITECTURE
            $architecturePreference = @()
            
            switch ($architecture) {
                "AMD64" { $architecturePreference = @("x64", "x86") }
                "ARM64" { $architecturePreference = @("arm64", "x64", "x86") }
                "x86"   { $architecturePreference = @("x86") }
                default { $architecturePreference = @("x64", "x86", "arm64") }
            }
            
            Write-Verbose "Detected architecture: $architecture. Searching in order: $($architecturePreference -join ', ')"

            $sdkRoots = @(
                "${env:ProgramFiles(x86)}\Windows Kits\10\bin",
                "${env:ProgramFiles(x86)}\Microsoft SDKs\Windows",
                "${env:ProgramFiles(x86)}\Microsoft Visual Studio\"
            )

            foreach ($root in $sdkRoots) {
                if (Test-Path $root) {
                    # First try architecture-specific paths
                    foreach ($arch in $architecturePreference) {
                        $archSpecificPaths = Get-ChildItem -Path $root -Directory -Recurse -ErrorAction SilentlyContinue | Where-Object { $_.Name -eq $arch }
                        foreach ($archPath in $archSpecificPaths) {
                            $toolPath = Join-Path $archPath.FullName $ToolName
                            if (Test-Path $toolPath -PathType Leaf) {
                                $foundTool = $toolPath
                                Write-Host "Found $ToolName for $arch architecture: $foundTool" -ForegroundColor Green
                                break
                            }
                        }
                        if ($foundTool) { break }
                    }
                    
                    # If not found in architecture-specific paths, fall back to general search
                    if (-not $foundTool) {
                        $foundToolPath = Get-ChildItem -Path $root -Filter $ToolName -Recurse -File -ErrorAction SilentlyContinue | Sort-Object LastWriteTime -Descending | Select-Object -ExpandProperty FullName -First 1
                        if ($foundToolPath) {
                            $foundTool = $foundToolPath
                            Write-Host "Found $ToolName in SDK/VS installations: $foundTool" -ForegroundColor Green
                            break
                        }
                    }
                }
                if ($foundTool) { break }
            }
        }
    }

    return $foundTool
}

function Cleanup-IntermediateFiles {
    param(
        [string]$OutputDir,
        [string]$ManifestBaseName
    )
    Write-Host "`nCleaning up intermediate files..." -ForegroundColor Cyan

    $intermediateExtensions = @(".h", ".rc", ".bin", ".res", ".exp", ".lib", ".obj", ".pdb", ".mc.out", ".mc.err", ".rc.out", ".rc.err", ".link.out", ".link.err")

    foreach ($ext in $intermediateExtensions) {
        $filePath = Join-Path $OutputDir "$ManifestBaseName$ext"
        if (Test-Path $filePath -PathType Leaf) {
            Remove-Item $filePath -Force -ErrorAction SilentlyContinue
            Write-Verbose "Removed: $filePath"
        }
    }
    
    # Clean up MSG*.bin files
    $msgBinFiles = Get-ChildItem -Path $OutputDir -Filter "MSG*.bin" -ErrorAction SilentlyContinue
    foreach ($file in $msgBinFiles) {
        Remove-Item $file.FullName -Force -ErrorAction SilentlyContinue
        Write-Verbose "Removed: $($file.FullName)"
    }
    
    # Clean up sampleTEMP.bin file
    $tempBinFile = Join-Path $OutputDir "$ManifestBaseName`TEMP.BIN"
    if (Test-Path $tempBinFile -PathType Leaf) {
        Remove-Item $tempBinFile -Force -ErrorAction SilentlyContinue
        Write-Verbose "Removed: $tempBinFile"
    }
    
    Write-Host "Cleanup complete." -ForegroundColor Green
}

if (-not (Test-Path $ManifestFile -PathType Leaf)) {
    Write-Error "Manifest file '$ManifestFile' not found. Please provide a valid path."
    exit 1
}

$ManifestPath = (Resolve-Path $ManifestFile).Path
$OutputDir = (Resolve-Path $OutputPath).Path

$ManifestBaseName = (Get-Item $ManifestPath).BaseName
$OutputDllName = "$ManifestBaseName.dll"

Write-Host "Locating required compilation tools..." -ForegroundColor DarkYellow

$mcExe = Find-Tool -ToolName "mc.exe" -ProvidedPath $McExePath -EnvVarName "MC_EXE_PATH"
if ($mcExe -eq $null) {
    Write-Error "Message Compiler (mc.exe) not found. See error message above for installation instructions."
    exit 1
}

$rcExe = Find-Tool -ToolName "rc.exe" -ProvidedPath $RcExePath -EnvVarName "RC_EXE_PATH"
if ($rcExe -eq $null) {
    Write-Error "Resource Compiler (rc.exe) not found. This is part of the Windows SDK. Please install it."
    exit 1
}

$linkExe = Find-Tool -ToolName "link.exe" -ProvidedPath $LinkExePath -EnvVarName "LINK_EXE_PATH"
if ($linkExe -eq $null) {
    Write-Error "Linker (link.exe) not found. This is part of Visual Studio Build Tools or Windows SDK. Please install it."
    exit 1
}

Write-Host "`n--- Starting ETW Manifest Compilation ---" -ForegroundColor Blue
Write-Host "  Manifest: $ManifestPath"
Write-Host "  Output Directory: $OutputDir"
Write-Host "  Target DLL: $OutputDllName"

try {
    Write-Host "`nStep 1 of 3: Compiling manifest with mc.exe..." -ForegroundColor Green
    $mcArgs = @(
        "-um",
        "-U",
        "-r", $OutputDir,
        $ManifestPath
    )
    
    $mcProcessOut = Join-Path $OutputDir "$ManifestBaseName.mc.out"
    $mcProcessErr = Join-Path $OutputDir "$ManifestBaseName.mc.err"

    Write-Verbose "Running: `"$mcExe`" $($mcArgs -join ' ')"
    $mcResult = Start-Process -FilePath $mcExe -ArgumentList $mcArgs -Wait -NoNewWindow -PassThru -RedirectStandardOutput $mcProcessOut -RedirectStandardError $mcProcessErr
    $mcResult | Wait-Process -ErrorAction SilentlyContinue

    if ($mcResult.ExitCode -ne 0) {
        $mcErrorOutput = Get-Content $mcProcessErr -ErrorAction SilentlyContinue
        $errorMessage = "mc.exe failed with exit code: $($mcResult.ExitCode)"
        if ($mcErrorOutput) { 
            $errorMessage += "`nmc.exe Error Output:`n$($mcErrorOutput -join "`n")"
        }
        Write-Error $errorMessage
        exit 1
    }
    Write-Host "✓ mc.exe compilation successful. Generated .h, .rc, .bin files." -ForegroundColor Green

    Write-Host "`nStep 2 of 3: Compiling resources with rc.exe..." -ForegroundColor Green
    $rcFile = Join-Path $OutputDir "$ManifestBaseName.rc"
    $resFile = Join-Path $OutputDir "$ManifestBaseName.res"
    
    if (-not (Test-Path $rcFile -PathType Leaf)) {
        Write-Error "Resource file '$rcFile' not found after mc.exe. Cannot proceed."
        exit 1
    }

    $rcArgs = @(
        "/r",
        "/fo", $resFile,
        $rcFile
    )

    $rcProcessOut = Join-Path $OutputDir "$ManifestBaseName.rc.out"
    $rcProcessErr = Join-Path $OutputDir "$ManifestBaseName.rc.err"

    Write-Verbose "Running: `"$rcExe`" $($rcArgs -join ' ')"
    $rcResult = Start-Process -FilePath $rcExe -ArgumentList $rcArgs -Wait -NoNewWindow -PassThru -RedirectStandardOutput $rcProcessOut -RedirectStandardError $rcProcessErr
    $rcResult | Wait-Process -ErrorAction SilentlyContinue

    if ($rcResult.ExitCode -ne 0) {
        $rcErrorOutput = Get-Content $rcProcessErr -ErrorAction SilentlyContinue
        $errorMessage = "rc.exe failed with exit code: $($rcResult.ExitCode)"
        if ($rcErrorOutput) { 
            $errorMessage += "`nrc.exe Error Output:`n$($rcErrorOutput -join "`n")"
        }
        Write-Error $errorMessage
        exit 1
    }
    Write-Host "✓ rc.exe compilation successful. Generated .res file." -ForegroundColor Green

    Write-Host "`nStep 3 of 3: Linking to create DLL with link.exe..." -ForegroundColor Green
    $outputDllPath = Join-Path $OutputDir $OutputDllName

    $linkArgs = @(
        "/DLL",
        "/NOENTRY",
        "/OUT:$outputDllPath",
        $resFile
    )

    $linkProcessOut = Join-Path $OutputDir "$ManifestBaseName.link.out"
    $linkProcessErr = Join-Path $OutputDir "$ManifestBaseName.link.err"

    Write-Verbose "Running: `"$linkExe`" $($linkArgs -join ' ')"
    $linkResult = Start-Process -FilePath $linkExe -ArgumentList $linkArgs -Wait -NoNewWindow -PassThru -RedirectStandardOutput $linkProcessOut -RedirectStandardError $linkProcessErr
    $linkResult | Wait-Process -ErrorAction SilentlyContinue

    if ($linkResult.ExitCode -ne 0) {
        $linkErrorOutput = Get-Content $linkProcessErr -ErrorAction SilentlyContinue
        $errorMessage = "link.exe failed with exit code: $($linkResult.ExitCode)"
        if ($linkErrorOutput) { 
            $errorMessage += "`nlink.exe Error Output:`n$($linkErrorOutput -join "`n")"
        }
        Write-Error $errorMessage
        exit 1
    }
    Write-Host "✓ link.exe successful. Created '$OutputDllName'." -ForegroundColor Green

    Write-Host "`n--- ETW Manifest Compilation to DLL Completed Successfully! ---" -ForegroundColor Green
    Write-Host "Output DLL: $outputDllPath" -ForegroundColor Green

} catch {
    Write-Error "An unexpected error occurred during the compilation process: $($_.Exception.Message)"
    exit 1
} finally {
    Cleanup-IntermediateFiles -OutputDir $OutputDir -ManifestBaseName $ManifestBaseName
}
