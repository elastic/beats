### Documentation
# This is a Vagrantfile for Beats development.
#
# Boxes
# =====
#
# win2012
# -------
# This box is used as a Windows development and testing environment for Beats.
#
# Usage and Features:
#   - Two users exist: Administrator and Vagrant. Both have the password: vagrant
#   - Use 'vagrant ssh' to open a Windows command prompt.
#   - Use 'vagrant rdp' to open a Windows Remote Desktop session. Mac users must
#     install the Microsoft Remote Desktop Client from the App Store.
#   - There is a desktop shortcut labeled "Beats Shell" that opens a command prompt
#     to C:\Gopath\src\github.com\elastic\beats where the code is mounted.
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

GO_VERSION = File.read(File.join(File.dirname(__FILE__), ".go-version")).strip

# Provisioning for Windows PowerShell
$winPsProvision = <<SCRIPT
echo 'Creating github.com\elastic in the GOPATH'
New-Item -itemtype directory -path "C:\\Gopath\\src\\github.com\\elastic" -force
echo "Symlinking C:\\Vagrant to C:\\Gopath\\src\\github.com\\elastic"
cmd /c mklink /d C:\\Gopath\\src\\github.com\\elastic\\beats \\\\vboxsvr\\vagrant

echo "Installing gvm to manage go version"
[Net.ServicePointManager]::SecurityProtocol = "tls12"
Invoke-WebRequest -URI https://github.com/andrewkroh/gvm/releases/download/v0.1.0/gvm-windows-amd64.exe -Outfile C:\Windows\System32\gvm.exe
C:\Windows\System32\gvm.exe --format=powershell #{GO_VERSION} | Invoke-Expression
go version

echo "Configure environment variables"
[System.Environment]::SetEnvironmentVariable("GOROOT", "C:\\Users\\vagrant\\.gvm\\versions\\go#{GO_VERSION}.windows.amd64", [System.EnvironmentVariableTarget]::Machine)
[System.Environment]::SetEnvironmentVariable("PATH", "$env:GOROOT\\bin;$env:PATH", [System.EnvironmentVariableTarget]::Machine)

echo "Creating Beats Shell desktop shortcut"
$WshShell = New-Object -comObject WScript.Shell
$Shortcut = $WshShell.CreateShortcut("$Home\\Desktop\\Beats Shell.lnk")
$Shortcut.TargetPath = "cmd.exe"
$Shortcut.Arguments = '/c "SET GOROOT=C:\\Users\\vagrant\\.gvm\\versions\\go#{GO_VERSION}.windows.amd64&PATH=C:\\Users\\vagrant\\.gvm\\versions\\go#{GO_VERSION}.windows.amd64\\bin;%PATH%" && START'
$Shortcut.WorkingDirectory = "C:\\Gopath\\src\\github.com\\elastic\\beats"
$Shortcut.Save()

echo "Disable automatic updates"
$AUSettings = (New-Object -com "Microsoft.Update.AutoUpdate").Settings
$AUSettings.NotificationLevel = 1
$AUSettings.Save()
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

Vagrant.configure(2) do |config|

  # Windows Server 2012 R2
  config.vm.define "win2012", primary: true do |win2012|

    win2012.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-win2012-r2-virtualbox-2016-10-28_1224.box"
    win2012.vm.guest = :windows

    # Communicator for windows boxes
    win2012.vm.communicator = "winrm"

    # Port forward WinRM and RDP
    win2012.vm.network :forwarded_port, guest: 22,   host: 2222,  id: "ssh", auto_correct: true
    win2012.vm.network :forwarded_port, guest: 3389, host: 33389, id: "rdp", auto_correct: true
    win2012.vm.network :forwarded_port, guest: 5985, host: 55985, id: "winrm", auto_correct: true

    win2012.vm.provision "shell", inline: $winPsProvision
  end

  # Solaris 11.2
  config.vm.define "solaris", primary: true do |solaris|
    solaris.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-solaris-11.2-virtualbox-2016-11-02_1603.box"
    solaris.vm.network :forwarded_port, guest: 22,   host: 2223,  id: "ssh", auto_correct: true

    solaris.vm.provision "shell", inline: $unixProvision, privileged: false
  end

  # FreeBSD 11.0
  config.vm.define "freebsd", primary: true do |freebsd|
    freebsd.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-freebsd-11.0-virtualbox-2016-11-02_1638.box"
    freebsd.vm.network :forwarded_port, guest: 22,   host: 2224,  id: "ssh", auto_correct: true

    # Must use NFS to sync a folder on FreeBSD and this requires a host-only network.
    # To enable the /vagrant folder, set disabled to false and uncomment the private_network.
    config.vm.synced_folder ".", "/vagrant", id: "vagrant-root", :nfs => true, disabled: true
    #config.vm.network "private_network", ip: "192.168.135.18"

    freebsd.vm.hostname = "beats-tester"
    freebsd.vm.provision "shell", inline: $unixProvision, privileged: false
  end

  # OpenBSD 5.9-stable
  config.vm.define "openbsd", primary: true do |openbsd|
    openbsd.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-openbsd-5.9-current-virtualbox-2016-11-02_2007.box"
    openbsd.vm.network :forwarded_port, guest: 22,   host: 2225,  id: "ssh", auto_correct: true

    config.vm.synced_folder ".", "/vagrant", type: "rsync", disabled: true
    config.vm.provider :virtualbox do |vbox|
      vbox.check_guest_additions = false
      vbox.functional_vboxsf = false
    end

    openbsd.vm.provision "shell", inline: $unixProvision, privileged: false
  end

  config.vm.define "precise64", primary: true do |c|
    c.vm.box = "ubuntu/precise64"
    c.vm.network :forwarded_port, guest: 22,   host: 2226,  id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false

    c.vm.synced_folder ".", "/vagrant", type: "virtualbox"
  end

  config.vm.define "precise32", primary: true do |c|
    c.vm.box = "ubuntu/precise32"
    c.vm.network :forwarded_port, guest: 22,   host: 2226,  id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision("386"), privileged: false

    c.vm.synced_folder ".", "/vagrant", type: "virtualbox"
  end

  config.vm.define "centos6", primary: true do |c|
    c.vm.box = "bento/centos-6.9"
    c.vm.network :forwarded_port, guest: 22,   host: 2229,  id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc python-pip python-virtualenv git"

    c.vm.synced_folder ".", "/vagrant", type: "virtualbox"
  end

  config.vm.define "fedora27", primary: true do |c|
    c.vm.box = "bento/fedora-27"
    c.vm.network :forwarded_port, guest: 22,   host: 2227,  id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: "dnf install -y make gcc python-pip python-virtualenv git"

    c.vm.synced_folder ".", "/vagrant", type: "virtualbox"
  end

  config.vm.define "archlinux", primary: true do |c|
    c.vm.box = "archlinux/archlinux"
    c.vm.network :forwarded_port, guest: 22,   host: 2228,  id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: "pacman -Sy && pacman -S --noconfirm make gcc python-pip python-virtualenv git"

    c.vm.synced_folder ".", "/vagrant", type: "virtualbox"
  end

  config.vm.define "ubuntu1804", primary: true do |c|
    c.vm.box = "ubuntu/bionic64"
    c.vm.network :forwarded_port, guest: 22,   host: 2229,  id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: "apt-get update && apt-get install -y make gcc python-pip python-virtualenv git"

    c.vm.synced_folder ".", "/vagrant", type: "virtualbox"
  end

  config.vm.define "sles12", primary: true do |c|
    c.vm.box = "elastic/sles-12-x86_64"
    c.vm.network :forwarded_port, guest: 22,   host: 2230,  id: "ssh", auto_correct: true

    c.vm.provision "shell", inline: $unixProvision, privileged: false
    c.vm.provision "shell", inline: linuxGvmProvision, privileged: false
    c.vm.provision "shell", inline: "pip install virtualenv"

    c.vm.synced_folder ".", "/vagrant", type: "virtualbox"
  end

  # Windows Server 2016
  config.vm.define "win2016", primary: true do |machine|
    machine.vm.box = "elastic/windows-2016-x86_64"
    machine.vm.provision "shell", inline: $winPsProvision

    machine.vm.provider "virtualbox" do |v|
      v.memory = 4096
    end
  end

end

# -*- mode: ruby -*-
# vi: set ft=ruby :
