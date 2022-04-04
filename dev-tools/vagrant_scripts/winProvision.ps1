param([String]$go_version)

Write-Host "Got go version: : " $go_version

$gopath_beats = "C:\\Gopath\\src\\github.com\\elastic\\beats"
if (-Not (Test-Path $gopath_beats)) {
    echo 'Creating github.com\\elastic in the GOPATH'
    New-Item -itemtype directory -path "C:\\Gopath\\src\\github.com\\elastic" -force
    echo "Symlinking C:\\Vagrant to C:\\Gopath\\src\\github.com\\elastic"
    cmd /c mklink /d $gopath_beats \\vboxsvr\vagrant
}

if (-Not (Get-Command "gvm" -ErrorAction SilentlyContinue)) {
    echo "Installing gvm to manage go version"
    [Net.ServicePointManager]::SecurityProtocol = "tls12"
    Invoke-WebRequest -URI https://github.com/andrewkroh/gvm/releases/download/v0.3.0/gvm-windows-amd64.exe -Outfile C:\\Windows\\System32\\gvm.exe
    C:\\Windows\\System32\\gvm.exe --format=powershell $go_version | Invoke-Expression
    go version

    echo "Configure Go environment variables"
    [System.Environment]::SetEnvironmentVariable("GOPATH", "C:\\Gopath", [System.EnvironmentVariableTarget]::Machine)
    [System.Environment]::SetEnvironmentVariable("GOROOT", "C:\\Users\\vagrant\\.gvm\\versions\\go$go_version.windows.amd64", [System.EnvironmentVariableTarget]::Machine)
    [System.Environment]::SetEnvironmentVariable("PATH", "%GOROOT%\\bin;$env:PATH;C:\\Gopath\\bin", [System.EnvironmentVariableTarget]::Machine)
}

$shell_link = "$Home\\Desktop\\Beats Shell.lnk"
if (-Not (Test-Path $shell_link)) {
    echo "Creating Beats Shell desktop shortcut"
    $WshShell = New-Object -comObject WScript.Shell
    $Shortcut = $WshShell.CreateShortcut($shell_link)
    $Shortcut.TargetPath = "powershell.exe"
    $Shortcut.Arguments = "-noexit -command '$gopath_beats'"
    $Shortcut.WorkingDirectory = $gopath_beats
    $Shortcut.Save()
}

Try {
    echo "Disabling automatic updates"
    $AUSettings = (New-Object -com "Microsoft.Update.AutoUpdate").Settings
    $AUSettings.NotificationLevel = 1
    $AUSettings.Save()
} Catch {
    echo "Failed to disable automatic updates."
}

if (-Not (Get-Command "choco" -ErrorAction SilentlyContinue)) {
    Set-ExecutionPolicy Bypass -Scope Process -Force
    iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
}

choco feature disable -n=showDownloadProgress

if (-Not (Get-Command "python" -ErrorAction SilentlyContinue)) {
    echo "Installing python 3"
    choco install python -y -r --version 3.8.2
    refreshenv
    $env:PATH = "$env:PATH;C:\\Python38;C:\\Python38\\Scripts"
}

echo "Updating pip"
python -m pip install --upgrade pip 2>&1 | %{ "$_" }

if (-Not (Get-Command "git" -ErrorAction SilentlyContinue)) {
    echo "Installing git"
    choco install git -y -r
}

if (-Not (Get-Command "gcc" -ErrorAction SilentlyContinue)) {
    echo "Installing mingw (gcc)"
    choco install mingw -y -r
}

echo "Setting PYTHON_ENV in VM to point to C:\\beats-python-env."
[System.Environment]::SetEnvironmentVariable("PYTHON_ENV", "C:\\beats-python-env", [System.EnvironmentVariableTarget]::Machine)
