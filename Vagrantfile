# -*- mode: ruby -*-
# vi: set ft=ruby :

### Documentation
# This box is used as windows development and testing environment for libbeat
# Two users exist: Administartor and Vagrant. Both have the password: vagrant

Vagrant.configure(2) do |config|

  # Windows Server 2012 R2
  config.vm.box = "http://files.ruflin.com/vagrant/beats-20150925.box"

  # Communicator for windows boxes
  config.vm.communicator = "winrm"
  #config.winrm.username = "admin"
  #config.winrm.password = "beats"

  # Port forward WinRM and RDP
  config.vm.network :forwarded_port, guest: 22, host: 22
  config.vm.network :forwarded_port, guest: 3389, host: 3389
  config.vm.network :forwarded_port, guest: 5985, host: 5985, id: "winrm", auto_correct: true

  # Load current folder as C:\dev directory
  config.vm.synced_folder ".", "/Gopath/src/github.com/elastic/libbeat"

end
