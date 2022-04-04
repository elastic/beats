#!/usr/bin/env bash

# This installs k8s utilities on the host
# This is the one command that needs to run as the user, hence the ugly sudo invocation
sudo -S -u vagrant -i /bin/bash -l -c "GO111MODULE='on' go get sigs.k8s.io/kind@v0.11.1"

curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg
echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | tee /etc/apt/sources.list.d/kubernetes.list
apt-get update
apt-get install -y kubectl