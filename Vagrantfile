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

TEST_BOXES = [
  {:name => "centos6", :box => "bento/centos-6.10", :platform => "centos", :extras => "yum install -y epel-release"},
  {:name => "centos7", :box => "bento/centos-7", :platform => "centos"},
  {:name => "centos8", :box => "bento/centos-7", :platform => "centos"},

  {:name => "win2012", :box => "https://s3.amazonaws.com/beats-files/vagrant/beats-win2012-r2-virtualbox-2016-10-28_1224.box", :platform => "windows"},
  {:name => "win2016", :box => "StefanScherer/windows_2016", :platform => "windows"},
  {:name => "win2019", :box => "StefanScherer/windows_2019", :platform => "windows"},

  {:name => "ubuntu1404", :box => "ubuntu/trusty64", :platform => "ubuntu"},
  {:name => "ubuntu1604", :box => "ubuntu/xenial64", :platform => "ubuntu"},
  {:name => "ubuntu1804", :box => "ubuntu/bionic64", :platform => "ubuntu"},
  {:name => "ubuntu2004", :box => "ubuntu/focal64", :platform => "ubuntu"},
]


#####
# Centos
#####
Vagrant.configure("2") do |config|
  config.vm.provider :virtualbox do |vbox|
    vbox.memory = 8192
    vbox.cpus = 6
  end

  # Docker config. Run with --provision-with docker
  # For now this script is only going to work on the ubuntu images.
  config.vm.provision "docker", type: "shell", run: "never" do |s|
    s.path = "dev-tools/vagrant_scripts/dockerProvision.sh"
  end

  config.vm.provision "kind", type: "shell", run: "never" do |s|
    s.path = "dev-tools/vagrant_scripts/kindProvision.sh"
  end


  TEST_BOXES.each_with_index do |node, idx|
    config.vm.define node[:name] do |nodeconfig|
      nodeconfig.vm.box = node[:box]
      nodeconfig.vm.network :forwarded_port, guest: 22, host: 2220 + idx, id: "ssh", auto_correct: true
      if node.has_key?(:extras)
        nodeconfig.vm.provision "shell", type: "shell", inline: node[:extras]
      end

      if node[:platform] == "centos" or node[:platform] == "ubuntu" or node[:platform] == "debian"
        nodeconfig.vm.provision "shell", type: "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
        nodeconfig.vm.provision "shell", type: "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
      end

      if node[:platform] == "centos"
        nodeconfig.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "rhel"
      end

      if node[:platform] == "ubuntu" or node[:platform] == "debian"
        nodeconfig.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "debian"
      end

      if node[:platform] == "windows"
        nodeconfig.vm.guest = :windows
        # Communicator for windows boxes
        nodeconfig.vm.communicator = "winrm"
        # Port forward WinRM and RDP
        nodeconfig.vm.network :forwarded_port, guest: 3389, host: 33389, id: "rdp", auto_correct: true
        nodeconfig.vm.network :forwarded_port, guest: 5985, host: 55985, id: "winrm", auto_correct: true
      end

    end
  end

end


Vagrant.configure("2") do |config|
  config.vm.provider :virtualbox do |vbox|
    vbox.memory = 8192
    vbox.cpus = 6
  end

  config.vm.define "debian8" do |c|
    c.vm.box = "generic/debian8"
    c.vm.network :forwarded_port, guest: 22, host: 2231, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "debian"
  end

  config.vm.define "debian9" do |c|
    c.vm.box = "debian/stretch64"
    c.vm.network :forwarded_port, guest: 22, host: 2232, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "debian"
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
   
  end

  config.vm.define "debian10" do |c|
    c.vm.box = "debian/buster64"
    c.vm.network :forwarded_port, guest: 22, host: 2233, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "debian"
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
    config.vm.provision "docker", type: "shell", run: "never" do |s|
      s.path = "dev-tools/vagrant_scripts/dockerProvision.sh"
    end
  end

  config.vm.define "amazon1" do |c|
    c.vm.box = "mvbcoding/awslinux"
    c.vm.network :forwarded_port, guest: 22, host: 2234, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc python3 python3-pip git rpm-devel"
  end

  config.vm.define "amazon2" do |c|
    c.vm.box = "bento/amazonlinux-2"
    c.vm.network :forwarded_port, guest: 22, host: 2235, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
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
    c.vm.provision "shell", inline: "pkg install -y -q bash"
    c.vm.provision "shell", inline: "chsh -s bash vagrant"
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 freebsd #{GO_VERSION}", privileged: false
    c.vm.provision "shell", inline: "sudo mount -t linprocfs /dev/null /proc", privileged: false
  end

  # OpenBSD 6.0
  config.vm.define "openbsd", autostart: false do |c|
    c.vm.box = "generic/openbsd6"
    c.vm.network :forwarded_port, guest: 22, host: 2238, id: "ssh", auto_correct: true

    c.vm.synced_folder ".", "/vagrant", type: "rsync", rsync__exclude: ".git/"
    c.vm.provider :virtualbox do |vbox|
      vbox.check_guest_additions = false
      vbox.functional_vboxsf = false
    end

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", inline: "sudo pkg_add go", privileged: true
  end

  config.vm.define "archlinux", autostart: false do |c|
    c.vm.box = "archlinux/archlinux"
    c.vm.network :forwarded_port, guest: 22, host: 2239, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 freebsd #{GO_VERSION}", privileged: false
    c.vm.provision "shell", inline: "pacman -Sy && pacman -S --noconfirm make gcc python python-pip git"
  end
end
