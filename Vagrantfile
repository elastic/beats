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
#   - Two users exist: Administartor and Vagrant. Both have the password: vagrant
#   - Use 'vagrant ssh' to open a Windows command prompt.
#   - Use 'vagrant rdp' to open a Windows Remote Deskop session. Mac users must
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

# Provisioning for Windows PowerShell
$winPsProvision = <<SCRIPT
echo 'Creating github.com\elastic in the GOPATH'
New-Item -itemtype directory -path "C:\\Gopath\\src\\github.com\\elastic" -force
echo "Symlinking C:\\Vagrant to C:\\Gopath\\src\\github.com\\elastic"
cmd /c mklink /d C:\\Gopath\\src\\github.com\\elastic\\beats \\\\vboxsvr\\vagrant

echo "Creating Beats Shell desktop shortcut"
$WshShell = New-Object -comObject WScript.Shell
$Shortcut = $WshShell.CreateShortcut("$Home\\Desktop\\Beats Shell.lnk")
$Shortcut.TargetPath = "cmd.exe"
$Shortcut.Arguments = "/K cd /d C:\\Gopath\\src\\github.com\\elastic\\beats"
$Shortcut.Save()

echo "Disable automatic updates"
$AUSettigns = (New-Object -com "Microsoft.Update.AutoUpdate").Settings
$AUSettigns.NotificationLevel = 1
$AUSettigns.Save()
SCRIPT

# Provisioning for Unix/Linux
$unixProvision = <<SCRIPT
echo 'Creating github.com/elastic in the GOPATH'
mkdir -p ~/go/src/github.com/elastic
echo 'Symlinking /vagrant to ~/go/src/github.com/elastic'
cd ~/go/src/github.com/elastic
if [ -d "/vagrant" ]; then ln -s /vagrant beats; fi
SCRIPT

Vagrant.configure(2) do |config|

  # Windows Server 2012 R2
  config.vm.define "win2012", primary: true do |win2012|

    win2012.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-win2012-r2-virtualbox-2016-01-20_0057.box"
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
    solaris.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-solaris-11.2-virtualbox-2016-01-23_0522.box"
    solaris.vm.network :forwarded_port, guest: 22,   host: 2223,  id: "ssh", auto_correct: true

    solaris.vm.provision "shell", inline: $unixProvision, privileged: false
  end

  # FreeBSD 11.0
  config.vm.define "freebsd", primary: true do |freebsd|
    freebsd.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-freebsd-11.0-virtualbox-2016-01-23_1919.box"
    freebsd.vm.network :forwarded_port, guest: 22,   host: 2224,  id: "ssh", auto_correct: true

    # Must use NFS to sync a folder on FreeBSD and this requires a host-only network.
    # To enable the /vagrant folder, set disabled to false and uncomment the private_network.
    config.vm.synced_folder ".", "/vagrant", id: "vagrant-root", :nfs => true, disabled: true
    #config.vm.network "private_network", ip: "192.168.135.18"

    freebsd.vm.provision "shell", inline: $unixProvision, privileged: false
  end

  # OpenBSD 5.9-current
  config.vm.define "openbsd", primary: true do |openbsd|
    openbsd.vm.box = "https://s3.amazonaws.com/beats-files/vagrant/beats-openbsd-5.9-current-virtualbox-2016-04-22_0422.box"
    openbsd.vm.network :forwarded_port, guest: 22,   host: 2225,  id: "ssh", auto_correct: true

    config.vm.synced_folder ".", "/vagrant", type: "rsync", disabled: true
    config.vm.provider :virtualbox do |vbox|
      vbox.check_guest_additions = false
      vbox.functional_vboxsf = false
    end

    openbsd.vm.provision "shell", inline: $unixProvision, privileged: false
  end

end

# -*- mode: ruby -*-
# vi: set ft=ruby :
