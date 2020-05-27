# Read the branch's Go version from the .go-version file.
GO_VERSION = File.read(File.join(File.dirname(__FILE__), ".go-version")).strip

create_symlink = <<SCRIPT
echo 'Creating github.com/elastic in the GOPATH'
mkdir -p ~/go/src/github.com/elastic
echo 'Symlinking /vagrant to ~/go/src/github.com/elastic'
cd ~/go/src/github.com/elastic
if [ -d "/vagrant" ]  && [ ! -e "go-libaudit" ]; then ln -s /vagrant go-libaudit; fi
SCRIPT

install_gvm = <<SCRIPT
mkdir -p ~/bin
if [ ! -e "~/bin/gvm" ]; then
  curl -sL -o ~/bin/gvm https://github.com/andrewkroh/gvm/releases/download/v0.2.2/gvm-linux-amd64
  chmod +x ~/bin/gvm
  ~/bin/gvm #{GO_VERSION}
  echo 'export GOPATH=$HOME/go' >> ~/.bash_profile
  echo 'export PATH=$HOME/bin:$GOPATH/bin:$PATH' >> ~/.bash_profile
  echo 'eval "$(gvm #{GO_VERSION})"' >> ~/.bash_profile
fi
SCRIPT

Vagrant.configure(2) do |config|
  config.vm.box = "ubuntu/bionic64"
  config.vm.network :forwarded_port, guest: 22, host: 2228, id: "ssh", auto_correct: true
  config.vm.provision "shell", inline: create_symlink, privileged: false
  config.vm.provision "shell", inline: install_gvm, privileged: false
  config.vm.provision "shell", inline: "apt-get update && apt-get install -y make gcc python3 python3-pip python3-venv git"
end
