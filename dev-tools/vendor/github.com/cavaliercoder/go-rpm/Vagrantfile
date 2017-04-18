# -*- mode: ruby -*-
# vi: set ft=ruby :

$script = <<end
# install build tools and runtime prerequisities
yum install -y epel-release
yum install -y \
	bzip2 \
	createrepo \
	git \
	golang \
	make \
	mercurial \
	sqlite \
	yum-utils

# configure GOPATH for user vagrant
mkdir /home/vagrant/go
cat >> /home/vagrant/.bashrc <<EOF
export GOPATH=\\$HOME/go
export PATH=\\$PATH:/vagrant:\\$HOME/go/bin

EOF

# fix perms
chown -R vagrant.vagrant /home/vagrant

end

Vagrant.configure(2) do |config|
  config.vm.box = "chef/centos-7.0"
  config.vm.provision "shell", inline: $script
end
