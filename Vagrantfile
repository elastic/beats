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
# add more boxes here
# -------------------
# More development boxes can be added to this file and you can run commands
# like "vagrant up solaris" or "vagrant up winxp" to start them.

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

Vagrant.configure(2) do |config|

  config.vm.define "win2012", primary: true do |win2012|
    # Windows Server 2012 R2
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

end

# -*- mode: ruby -*-
# vi: set ft=ruby :
