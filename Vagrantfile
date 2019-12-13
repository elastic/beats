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
#   - Folder syncing doesn't work well. Consider copying the files into the box or
#     cloning the project inside the box.
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
    Invoke-WebRequest -URI https://github.com/andrewkroh/gvm/releases/download/v0.2.1/gvm-windows-amd64.exe -Outfile C:\\Windows\\System32\\gvm.exe
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
    echo "Installing python2"
    choco install python2 -y -r
    refreshenv
    $env:PATH = "$env:PATH;C:\\Python27;C:\\Python27\\Scripts"
}

if (-Not (Get-Command "pip" -ErrorAction SilentlyContinue)) {
    echo "Installing pip"
    Invoke-WebRequest https://bootstrap.pypa.io/get-pip.py -OutFile get-pip.py
    python get-pip.py -U --force-reinstall 2>&1 | %{ "$_" }
    rm get-pip.py
    Invoke-WebRequest
} else {
    echo "Updating pip"
    python -m pip install --upgrade pip 2>&1 | %{ "$_" }
}

if (-Not (Get-Command "virtualenv" -ErrorAction SilentlyContinue)) {
    echo "Installing virtualenv"
    python -m pip install virtualenv 2>&1 | %{ "$_" }
}

if (-Not (Get-Command "git" -ErrorAction SilentlyContinue)) {
    echo "Installing git"
    choco install git -y -r
}

if (-Not (Get-Command "gcc" -ErrorAction SilentlyContinue)) {
    echo "Installing mingw (gcc)"
    choco install mingw -y -r
}
SCRIPT

# Provisioning for Unix/Linux
$unixProvision = <<SCRIPT
echo 'Creating github.com/elastic in the GOPATH'
mkdir -p ~/go/src/github.com/elastic
echo 'Symlinking /vagrant to ~/go/src/github.com/elastic'
cd ~/go/src/github.com/elastic
if [ -d "/vagrant" ]  && [ ! -e "beats" ]; then ln -s /vagrant beats; fi
SCRIPT

# Linux GVM
def linuxGvmProvision(arch="amd64")
  return <<SCRIPT
mkdir -p ~/bin
if [ ! -e "~/bin/gvm" ]; then
  curl -sL -o ~/bin/gvm https://github.com/andrewkroh/gvm/releases/download/v0.1.0/gvm-linux-#{arch}
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
apt-get install -y make gcc python-pip python-virtualenv git
SCRIPT
end

Vagrant.configure(2) do |config|
  # Windows Server 2012 R2
  config.vm.define "win2012", primary: true do |c|
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

  config.vm.define "win2016", primary: true do |c|
    c.vm.box = "StefanScherer/windows_2016"
    c.vm.provision "shell", inline: $winPsProvision, privileged: false
  end

  config.vm.define "win2019", primary: true do |c|
    c.vm.box = "StefanScherer/windows_2019"
    c.vm.provision "shell", inline: $winPsProvision, privileged: false
  end

  # Solaris 11.2
  config.vm.define "solaris", primary: true do |c|
    c.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-solaris-11.2-virtualbox-2016-11-02_1603.box"
    c.vm.network :forwarded_port, guest: 22, host: 2223, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
  end

  # FreeBSD 11.0
  config.vm.define "freebsd", primary: true do |c|
    c.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-freebsd-11.0-virtualbox-2016-11-02_1638.box"
    c.vm.network :forwarded_port, guest: 22, host: 2224, id: "ssh", auto_correct: true

    # Must use NFS to sync a folder on FreeBSD and this requires a host-only network.
    # To enable the /vagrant folder, set disabled to false and uncomment the private_network.
    c.vm.synced_folder ".", "/vagrant", id: "vagrant-root", :nfs => true, disabled: true
    #c.vm.network "private_network", ip: "192.168.135.18"

    c.vm.hostname = "beats-tester"
    c.vm.provision "shell", inline: $unixProvision, privileged: false
  end

  # OpenBSD 5.9-stable
  config.vm.define "openbsd", primary: true do |c|
    c.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-openbsd-5.9-current-virtualbox-2016-11-02_2007.box"
    c.vm.network :forwarded_port, guest: 22, host: 2225, id: "ssh", auto_correct: true

    c.vm.synced_folder ".", "/vagrant", type: "rsync", disabled: true
    c.vm.provider :virtualbox do |vbox|
      vbox.check_guest_additions = false
      vbox.functional_vboxsf = false
    end

    c.vm.provision "shell", inline: $unixProvision, privileged: false
  end

  config.vm.define "precise32", primary: true do |c|
    c.vm.box = "ubuntu/precise32"
    c.vm.network :forwarded_port, guest: 22, host: 2226, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision("386"), privileged: false
    c.vm.provision "shell", inline: linuxDebianProvision
  end

  config.vm.define "precise64", primary: true do |c|
    c.vm.box = "ubuntu/precise64"
    c.vm.network :forwarded_port, guest: 22, host: 2227, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: linuxDebianProvision
  end

  config.vm.define "ubuntu1804", primary: true do |c|
    c.vm.box = "ubuntu/bionic64"
    c.vm.network :forwarded_port, guest: 22, host: 2228, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: linuxDebianProvision
  end

  config.vm.define "centos6", primary: true do |c|
    c.vm.box = "bento/centos-6.10"
    c.vm.network :forwarded_port, guest: 22, host: 2229, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc python-pip python-virtualenv git rpm-devel"
  end

  config.vm.define "centos7", primary: true do |c|
    c.vm.box = "bento/centos-7"
    c.vm.network :forwarded_port, guest: 22, host: 2230, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc python-pip python-virtualenv git rpm-devel"
  end

  config.vm.define "fedora29", primary: true do |c|
    c.vm.box = "bento/fedora-29"
    c.vm.network :forwarded_port, guest: 22, host: 2231, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: "dnf install -y make gcc python-pip python-virtualenv git rpm-devel"
  end

  config.vm.define "sles12", primary: true do |c|
    c.vm.box = "elastic/sles-12-x86_64"
    c.vm.network :forwarded_port, guest: 22, host: 2232, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: "pip install virtualenv"
  end

  config.vm.define "archlinux", primary: true do |c|
    c.vm.box = "archlinux/archlinux"
    c.vm.network :forwarded_port, guest: 22, host: 2233, id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: "pacman -Sy && pacman -S --noconfirm make gcc python-pip python-virtualenv git"
  end
end
