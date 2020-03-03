# NOTE: This is not a public image. It's only available within the Elastic
# organization and requires a 'vagrant login'.

GO_VERSION = "1.12.4"

# Provisioning for Windows PowerShell.
$winPsProvision = <<SCRIPT
echo "Installing gvm to manage go version"
[Net.ServicePointManager]::SecurityProtocol = "tls12"
Invoke-WebRequest -URI https://github.com/andrewkroh/gvm/releases/download/v0.2.0/gvm-windows-amd64.exe -Outfile C:\Windows\System32\gvm.exe
C:\Windows\System32\gvm.exe --format=powershell #{GO_VERSION} | Invoke-Expression
go version

echo "Configure environment variables"
[System.Environment]::SetEnvironmentVariable("GOROOT", "C:\\Users\\vagrant\\.gvm\\versions\\go#{GO_VERSION}.windows.amd64", [System.EnvironmentVariableTarget]::Machine)
[System.Environment]::SetEnvironmentVariable("PATH", "$env:GOROOT\\bin;$env:PATH", [System.EnvironmentVariableTarget]::Machine)
SCRIPT

Vagrant.configure("2") do |config|
  config.vm.box = "elastic/windows-2016-x86_64"

  config.vm.provision "shell", inline: $winPsProvision

  config.vm.provider "virtualbox" do |v|
    v.memory = 4096
  end
end

# -*- mode: ruby -*-
# vi: set ft=ruby :
