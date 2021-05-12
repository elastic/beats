### Documentation
#
# This is a Vagrantfile for Beats development and testing. These are unofficial
# environments to help developers test things in different environments.
#
# Notes
# =====
#
# win2012, win2016, win2019
# -------------------------
#
# To login install Microsoft Remote Desktop Client (available in Mac App Store).
# Then run 'vagrant rdp' and login as user/pass vagrant/vagrant. Or you can
# manually configure your RDP client to connect to the mapped 3389 port as shown
# by 'vagrant port win2019'.
#
# The provisioning currently does no install libpcap sources or a pcap driver
# (like npcap) so Packetbeat will not build/run without some manually setup.
#
# solaris
# -------------------
#   - Use gmake instead of make.
#
# freebsd and openbsd
# -------------------
#   - Use gmake instead of make.
#   - Folder syncing doesn't work well. Consider copying the files into the box
#     or cloning the project inside the box.
###

# Read the branch's Go version from the .go-version file.
GO_VERSION = File.read(File.join(File.dirname(__FILE__), ".go-version")).strip

# Provisioning for Windows PowerShell
$winPsProvision = <<SCRIPT
$gopath_beats = "C:\\Gopath\\src\\github.com\\elastic\\beats"
if (-Not (Test-Path $gopath_beats)) {
    echo 'Creating github.com\\elastic in the GOPATH'
    New-Item -itemtype directory -path "C:\\Gopath\\src\\github.com\\elastic" -force
    echo "Symlinking C:\\Vagrant to C:\\Gopath\\src\\github.com\\elastic"
    cmd /c mklink /d $gopath_beats \\\\vboxsvr\\vagrant
}

if (-Not (Get-Command "gvm" -ErrorAction SilentlyContinue)) {
    echo "Installing gvm to manage go version"
    [Net.ServicePointManager]::SecurityProtocol = "tls12"
    Invoke-WebRequest -URI https://github.com/andrewkroh/gvm/releases/download/v0.3.0/gvm-windows-amd64.exe -Outfile C:\\Windows\\System32\\gvm.exe
    C:\\Windows\\System32\\gvm.exe --format=powershell #{GO_VERSION} | Invoke-Expression
    go version

    echo "Configure Go environment variables"
    [System.Environment]::SetEnvironmentVariable("GOPATH", "C:\\Gopath", [System.EnvironmentVariableTarget]::Machine)
    [System.Environment]::SetEnvironmentVariable("GOROOT", "C:\\Users\\vagrant\\.gvm\\versions\\go#{GO_VERSION}.windows.amd64", [System.EnvironmentVariableTarget]::Machine)
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
SCRIPT

# Provisioning for Unix/Linux
$unixProvision = <<SCRIPT
echo 'Creating github.com/elastic in the GOPATH'
mkdir -p ~/go/src/github.com/elastic
echo 'Symlinking /vagrant to ~/go/src/github.com/elastic'
cd ~/go/src/github.com/elastic
if [ -d "/vagrant" ]  && [ ! -e "beats" ]; then ln -s /vagrant beats; fi
SCRIPT

$freebsdShellUpdate = <<SCRIPT
pkg install -y -q bash
chsh -s bash vagrant
SCRIPT


# Linux GVM
def gvmProvision(arch="amd64", os="linux")
  return <<SCRIPT
mkdir -p ~/bin
if [ ! -e "~/bin/gvm" ]; then
  curl -sL -o ~/bin/gvm https://github.com/andrewkroh/gvm/releases/download/v0.3.0/gvm-#{os}-#{arch}
  chmod +x ~/bin/gvm
  ~/bin/gvm #{GO_VERSION}
  echo 'export GOPATH=$HOME/go' >> ~/.bash_profile
  echo 'export PATH=$HOME/bin:$GOPATH/bin:$PATH' >> ~/.bash_profile
  echo 'eval "$(gvm #{GO_VERSION})"' >> ~/.bash_profile
fi
SCRIPT
end

# Provision packages for Linux Debian.
def linuxDebianProvision()
  return <<SCRIPT
#!/usr/bin/env bash
set -eio pipefail
apt-get update
apt-get install -y make gcc python3 python3-pip python3-venv git libsystemd-dev
SCRIPT
end

Vagrant.configure("2") do |config|
  config.vm.provider :virtualbox do |vbox|
    vbox.memory = 4096
    vbox.cpus = 4
  end

  # Windows Server 2012 R2
  config.vm.define "win2012" do |c|
    c.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-win2012-r2-virtualbox-2016-10-28_1224.box"
    c.vm.guest = :windows

    # Communicator for windows boxes
    c.vm.communicator = "winrm"

    # Port forward WinRM and RDP
    c.vm.network :forwarded_port, guest: 22, host: 2222, id: "ssh", auto_correct: true
    c.vm.network :forwarded_port, guest: 3389, host: 33389, id: "rdp", auto_correct: true
    c.vm.network :forwarded_port, guest: 5985, host: 55985, id: "winrm", auto_correct: true

    c.vm.provision "shell", inline: $winPsProvision
  end

  config.vm.define "win2016" do |c|
    c.vm.box = "StefanScherer/windows_2016"
    c.vm.provision "shell", inline: $winPsProvision, privileged: false
  end

  config.vm.define "win2019" do |c|
    c.vm.box = "StefanScherer/windows_2019"
    c.vm.provision "shell", inline: $winPsProvision, privileged: false
  end

  config.vm.define "centos6" do |c|
    c.vm.box = "bento/centos-6.10"
    c.vm.network :forwarded_port, guest: 22, host: 2223, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc git rpm-devel epel-release"
    c.vm.provision "shell", inline: "yum install -y python34 python34-pip"
  end

  config.vm.define "centos7" do |c|
    c.vm.box = "bento/centos-7"
    c.vm.network :forwarded_port, guest: 22, host: 2224, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc python3 python3-pip git rpm-devel"
  end

  config.vm.define "centos8" do |c|
    c.vm.box = "bento/centos-8"
    c.vm.network :forwarded_port, guest: 22, host: 2225, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc python3 python3-pip git rpm-devel"
  end

  config.vm.define "ubuntu1404" do |c|
    c.vm.box = "ubuntu/trusty64"
    c.vm.network :forwarded_port, guest: 22, host: 2226, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: "apt-get update && apt-get install -y make gcc python3 python3-pip python3.4-venv git"
  end

  config.vm.define "ubuntu1604" do |c|
    c.vm.box = "ubuntu/xenial64"
    c.vm.network :forwarded_port, guest: 22, host: 2227, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: linuxDebianProvision
  end

  config.vm.define "ubuntu1804" do |c|
    c.vm.box = "ubuntu/bionic64"
    c.vm.network :forwarded_port, guest: 22, host: 2228, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: linuxDebianProvision
  end

  config.vm.define "ubuntu2004", primary: true  do |c|
    c.vm.box = "ubuntu/focal64"
    c.vm.network :forwarded_port, guest: 22, host: 2229, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: linuxDebianProvision
  end

  config.vm.define "debian8" do |c|
    c.vm.box = "debian/jessie64"
    c.vm.network :forwarded_port, guest: 22, host: 2231, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: linuxDebianProvision
  end

  config.vm.define "debian9" do |c|
    c.vm.box = "debian/stretch64"
    c.vm.network :forwarded_port, guest: 22, host: 2232, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: linuxDebianProvision
  end

  config.vm.define "debian10" do |c|
    c.vm.box = "debian/buster64"
    c.vm.network :forwarded_port, guest: 22, host: 2233, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: linuxDebianProvision
  end

  config.vm.define "amazon1" do |c|
    c.vm.box = "mvbcoding/awslinux"
    c.vm.network :forwarded_port, guest: 22, host: 2234, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc python3 python3-pip git rpm-devel"
  end

  config.vm.define "amazon2" do |c|
    c.vm.box = "bento/amazonlinux-2"
    c.vm.network :forwarded_port, guest: 22, host: 2235, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc python3 python3-pip git rpm-devel"
  end

  # The following boxes are not listed as officially supported by the Elastic support matrix
  # Solaris 11.2
  config.vm.define "solaris", autostart: false do |c|
    c.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-solaris-11.2-virtualbox-2016-11-02_1603.box"
    c.vm.network :forwarded_port, guest: 22, host: 2236, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
  end

  # FreeBSD 13.0
  config.vm.define "freebsd", autostart: false do |c|
    c.vm.box = "bento/freebsd-13"

    # Here Be Dragons: don't attempt to try and get nfs working, unless you have a lot of free time.
    # run `vagrant rsync-auto` to keep the host and guest in sync.
    c.vm.synced_folder ".", "/vagrant", type: "rsync", rsync__exclude: ".git/"

    c.vm.hostname = "beats-tester"
    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: $freebsdShellUpdate, privileged: true
    c.vm.provision "shell", inline: gvmProvision(arch="amd64", os="freebsd"), privileged: false
  end

  # OpenBSD 5.9-stable
  config.vm.define "openbsd", autostart: false do |c|
    c.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-openbsd-5.9-current-virtualbox-2016-11-02_2007.box"
    c.vm.network :forwarded_port, guest: 22, host: 2238, id: "ssh", auto_correct: true

    c.vm.synced_folder ".", "/vagrant", type: "rsync", disabled: true
    c.vm.provider :virtualbox do |vbox|
      vbox.check_guest_additions = false
      vbox.functional_vboxsf = false
    end

    c.vm.provision "shell", inline: $unixProvision, privileged: false
  end

  config.vm.define "archlinux", autostart: false do |c|
    c.vm.box = "archlinux/archlinux"
    c.vm.network :forwarded_port, guest: 22, host: 2239, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: gvmProvision, privileged: false
    c.vm.provision "shell", inline: "pacman -Sy && pacman -S --noconfirm make gcc python python-pip git"
  end
end
