function ArePathsChanged($patterns) {
    $changedlist = @()
    foreach ($pattern in $patterns) {
        $changedFiles = & git diff --name-only "HEAD@{1}" HEAD | Select-String -Pattern $pattern -SimpleMatch
        if ($changedFiles) {
            $changedlist += $changedFiles
        }
    }
    if ($changedlist) {
        Write-Host "--- Files changed: $changedlist"
        return $true
    }
    else {
        Write-Host "--- No files changed within specified changeset: $patterns"
        return $false
    }
}

function AreChangedOnlyPaths($patterns) {
    $changedFiles = & git diff --name-only "HEAD@{1}" HEAD
    Write-Host "--- Git Diff result:"
    Write-Host "$changedFiles"

    $matchedFiles = @()
    foreach ($pattern in $patterns) {
        $matched = $changedFiles | Select-String -Pattern $pattern -SimpleMatch
        if ($matched) {
            $matchedFiles += $matched
        }
    }
    if (($matchedFiles.Count -eq $changedFiles.Count) -or ($changedFiles.Count -eq 0)) {
        return $true
    }
    return $false
}

# This function sets a `MODULE` env var, required by IT tests, containing a comma separated list of modules for a given beats project (specified via the first argument).
# The list is built depending on directories that have changed under `modules/` excluding anything else such as asciidoc and png files.
# `MODULE` will empty if no changes apply.
function DefineModuleFromTheChangeSet($projectPath) {
    $projectPathTransformed = $projectPath -replace '/', '\\'
    $projectPathExclusion = "((?!^$projectPathTransformed\\\/).)*\$"
    $exclude = @("^($projectPathExclusion|((?!\\/module\\/).)*\$|.*\\.asciidoc|.*\\.png)")

    $changedModules = ''

    $moduleDirs = Get-ChildItem -Directory "$projectPath\module"
    foreach($moduleDir in $moduleDirs) {
        if((ArePathsChanged($moduleDir)) -and !(AreChangedOnlyPaths($exclude))) {
            if(!$changedModules) {
                $changedModules = $moduleDir.Name
            }
            else {
                $changedModules += ',' + $moduleDir.Name
            }
        }
    }

    if ($changedModules) {
        $env:MODULE = $changedModules
        Write-Output "~~~ Set env var MODULE to [$env:MODULE]"
        Write-Output "~~~ Resuming commands"
    }
}
