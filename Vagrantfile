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

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/winProvision.ps1", args: "-go_version #{GO_VERSION}"
  end

  config.vm.define "win2016" do |c|
    c.vm.box = "StefanScherer/windows_2016"
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/winProvision.ps1", args: "-go_version #{GO_VERSION}"
  end

  config.vm.define "win2019" do |c|
    c.vm.box = "StefanScherer/windows_2019"
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/winProvision.ps1", args: "-go_version #{GO_VERSION}"
  end

  config.vm.define "centos6" do |c|
    c.vm.box = "bento/centos-6.10"
    c.vm.network :forwarded_port, guest: 22, host: 2223, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc git rpm-devel epel-release"
    c.vm.provision "shell", inline: "yum install -y python34 python34-pip"
  end

  config.vm.define "centos7" do |c|
    c.vm.box = "bento/centos-7"
    c.vm.network :forwarded_port, guest: 22, host: 2224, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc python3 python3-pip git rpm-devel"
  end

  config.vm.define "centos8" do |c|
    c.vm.box = "bento/centos-8"
    c.vm.network :forwarded_port, guest: 22, host: 2225, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
    c.vm.provision "shell", inline: "yum install -y make gcc python3 python3-pip git rpm-devel"
  end

  config.vm.define "ubuntu1404" do |c|
    c.vm.box = "ubuntu/trusty64"
    c.vm.network :forwarded_port, guest: 22, host: 2226, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
    c.vm.provision "shell", inline: "apt-get update && apt-get install -y make gcc python3 python3-pip python3.4-venv git"
  end

  config.vm.define "ubuntu1604" do |c|
    c.vm.box = "ubuntu/xenial64"
    c.vm.network :forwarded_port, guest: 22, host: 2227, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "debian"

    config.vm.provision "docker", type: "shell", run: "never" do |s|
      s.path = "dev-tools/vagrant_scripts/dockerProvision.sh"
    end
  end

  config.vm.define "ubuntu1804" do |c|
    c.vm.box = "ubuntu/bionic64"
    c.vm.network :forwarded_port, guest: 22, host: 2228, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "debian"

    config.vm.provision "docker", type: "shell", run: "never" do |s|
      s.path = "dev-tools/vagrant_scripts/dockerProvision.sh"
    end
  end

  config.vm.define "ubuntu2004", primary: true  do |c|
    c.vm.box = "ubuntu/focal64"
    c.vm.network :forwarded_port, guest: 22, host: 2229, id: "ssh", auto_correct: true

    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
    c.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "debian"

    config.vm.provision "docker", type: "shell", run: "never" do |s|
      s.path = "dev-tools/vagrant_scripts/dockerProvision.sh"
    end
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
