#!/usr/bin/env bash

PROVISION_TYPE="$1"
ARCH="$2"
OS="$3"
GO_VERSION="$4"

function gvmProvision () {
    ARCH="$1"
    OS="$2"
    GO_VERSION="$3"
    echo "in gvmProvision: $ARCH / $OS"
    mkdir -p ~/bin
    if [ ! -e "~/bin/gvm" ]; then
        curl -sL -o ~/bin/gvm https://github.com/andrewkroh/gvm/releases/download/v0.3.0/gvm-$OS-$ARCH
        chmod +x ~/bin/gvm
        ~/bin/gvm $GO_VERSION
        echo 'export GOPATH=$HOME/go' >> ~/.bash_profile
        echo 'export PATH=$HOME/bin:$GOPATH/bin:$PATH' >> ~/.bash_profile
        source ~/.bash_profile
        gvm $GO_VERSION >> ~/.bash_profile
    fi
}

function unixProvision () {
    echo 'Creating github.com/elastic in the GOPATH'
    mkdir -p ~/go/src/github.com/elastic
    echo 'Symlinking /vagrant to ~/go/src/github.com/elastic'
    cd ~/go/src/github.com/elastic
    if [ -d "/vagrant" ]  && [ ! -e "beats" ]; then ln -s /vagrant beats; fi
}

function debProvision () {
    set -eo pipefail
    apt-get update
    apt-get install -y make gcc python3 python3-pip python3-venv git libsystemd-dev curl
}

function rhelProvision () {
    yum update
    yum install -y make gcc git python3 python3-pip python3-venv git rpm-devel
}

function archProvision () {
    pacman -Sy && pacman -S --noconfirm make gcc python python-pip git
}

function suseProvision() {
    zypper refresh
    zypper install -y make gcc git python3 python3-pip python3-virtualenv git rpm-devel
}

case "${PROVISION_TYPE}" in
    "gvm")
        gvmProvision $ARCH $OS $GO_VERSION
    ;;
    "archlinux")
        archProvision
    ;;
    "unix")
        unixProvision
    ;;
    "centos")
        rhelProvision
    ;;
    "debian" | "ubuntu")
        debProvision
    ;;
    "opensuse" | "sles")
        suseProvision
    ;;
    *)
        echo "No Extra provisioning steps for this platform"
esac
