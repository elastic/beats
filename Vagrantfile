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

  {:name => "rhel7", :box => "generic/rhel7", :platform => "redhat" },
  {:name => "rhel8", :box => "generic/rhel8", :platform => "redhat" },

  {:name => "win2012", :box => "https://s3.amazonaws.com/beats-files/vagrant/beats-win2012-r2-virtualbox-2016-10-28_1224.box", :platform => "windows"},
  {:name => "win2016", :box => "StefanScherer/windows_2016", :platform => "windows"},
  {:name => "win2019", :box => "StefanScherer/windows_2019", :platform => "windows"},

  {:name => "ubuntu1404", :box => "ubuntu/trusty64", :platform => "ubuntu"},
  {:name => "ubuntu1604", :box => "ubuntu/xenial64", :platform => "ubuntu"},
  {:name => "ubuntu1804", :box => "ubuntu/bionic64", :platform => "ubuntu"},
  {:name => "ubuntu2004", :box => "ubuntu/focal64", :platform => "ubuntu"},

  {:name => "debian8", :box => "generic/debian8", :platform => "debian"},
  {:name => "debian9", :box => "debian/stretch64", :platform => "debian"},
  {:name => "debian10", :box => "debian/buster64", :platform => "debian"},

  {:name => "amazon1", :box => "mvbcoding/awslinux", :platform => "centos"},
  {:name => "amazon2", :box => "bento/amazonlinux-2", :platform => "centos"},

  # Unsupported platforms
  {:name => "opensuse153", :box => "bento/opensuse-leap-15.3", :platform => "opensuse"},
  {:name => "sles12", :box => "elastic/sles-12-x86_64", :platform => "sles"},
  {:name => "solaris", :box => "https://s3.amazonaws.com/beats-files/vagrant/beats-solaris-11.2-virtualbox-2016-11-02_1603.box", :platform => "unix"},
  {:name => "freebsd", :box => "bento/freebsd-13", :platform => "freebsd", :extras => "pkg install -y -q bash && chsh -s bash vagrant"},
  {:name => "openbsd", :box => "generic/openbsd6", :platform => "openbsd", :extras => "sudo pkg_add go"},
  {:name => "arch", :box => "archlinux/archlinux", :platform => "archlinux", :extras => "pacman -Sy && pacman -S --noconfirm make gcc python python-pip git"},
]


Vagrant.configure("2") do |config|
  config.vm.provider :virtualbox do |vbox|
    vbox.memory = 8192
    vbox.cpus = 6
  end

  # Docker config. Run with --provision-with docker,shell
  # For now this script is only going to work on the ubuntu images.
  # How to run tests from within docker, from within the container:
  #  docker run -v $(pwd):"/root/go/src/github.com/elastic/beats" -w /root/go/src/github.com/elastic/beats/metricbeat/module/system/process --entrypoint="/usr/local/go/bin/go" -it docker.elastic.co/beats-dev/golang-crossbuild:1.16.6-darwin-debian10 test -v -tags=integrations -run TestFetch
  config.vm.provision "docker", type: "shell", run: "never" do |s|
    s.path = "dev-tools/vagrant_scripts/dockerProvision.sh"
  end

  config.vm.provision "kind", type: "shell", run: "never" do |s|
    s.path = "dev-tools/vagrant_scripts/kindProvision.sh"
  end


  # Loop to define boxes
  TEST_BOXES.each_with_index do |node, idx|
    config.vm.define node[:name] do |nodeconfig|
      nodeconfig.vm.box = node[:box]
      nodeconfig.vm.network :forwarded_port, guest: 22, host: 2220 + idx, id: "ssh", auto_correct: true
      if node.has_key?(:extras)
        nodeconfig.vm.provision "shell", inline: node[:extras]
      end

      if node[:platform] != "windows"
        nodeconfig.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "unix", privileged: false
        nodeconfig.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: node[:platform]
      end


      # for BSDs
      if node[:platform] == "openbsd" or node[:platform] == "freebsd"
        nodeconfig.vm.synced_folder ".", "/vagrant", type: "rsync", rsync__exclude: ".git/"
        nodeconfig.vm.provider :virtualbox do |vbox|
          vbox.check_guest_additions = false
          vbox.functional_vboxsf = false
        end
      end

      # Freebsd
      if node[:platform] == "freebsd"
        nodeconfig.vm.provision "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 freebsd #{GO_VERSION}", privileged: false
        nodeconfig.vm.provision "shell", inline: "sudo mount -t linprocfs /dev/null /proc", privileged: false
      end

      # gvm install
      if [:centos, :ubuntu, :debian, :archlinux, :opensuse, :sles, :redhat].include?(node[:platform].to_sym)
        nodeconfig.vm.provision "shell", type: "shell", path: "dev-tools/vagrant_scripts/unixProvision.sh", args: "gvm amd64 linux #{GO_VERSION}", privileged: false
      end

      if node[:platform] == "windows"
        nodeconfig.vm.guest = :windows
        nodeconfig.vm.provision "shell", path: "dev-tools/vagrant_scripts/winProvision.ps1", args: "#{GO_VERSION}"
        # Communicator for windows boxes
        nodeconfig.vm.communicator = "winrm"
        # Port forward WinRM and RDP
        nodeconfig.vm.network :forwarded_port, guest: 3389, host: 33389, id: "rdp", auto_correct: true
        nodeconfig.vm.network :forwarded_port, guest: 5985, host: 55985, id: "winrm", auto_correct: true
      end

    end
  end

end
