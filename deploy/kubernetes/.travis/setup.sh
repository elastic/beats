# This script assumes Docker is already installed
#!/bin/bash

set -x

# set docker0 to promiscuous mode
sudo ip link set docker0 promisc on

# install etcd
wget https://github.com/coreos/etcd/releases/download/$TRAVIS_ETCD_VERSION/etcd-$TRAVIS_ETCD_VERSION-linux-amd64.tar.gz
tar xzf etcd-$TRAVIS_ETCD_VERSION-linux-amd64.tar.gz
sudo mv etcd-$TRAVIS_ETCD_VERSION-linux-amd64/etcd /usr/local/bin/etcd
rm etcd-$TRAVIS_ETCD_VERSION-linux-amd64.tar.gz
rm -rf etcd-$TRAVIS_ETCD_VERSION-linux-amd64

# download kubectl
wget https://storage.googleapis.com/kubernetes-release/release/$TRAVIS_KUBE_VERSION/bin/linux/amd64/kubectl
chmod +x kubectl
sudo mv kubectl /usr/local/bin/kubectl

# download kubernetes
git clone https://github.com/kubernetes/kubernetes $HOME/kubernetes

# install cfssl
go get -u github.com/cloudflare/cfssl/cmd/...

pushd $HOME/kubernetes
  git checkout $TRAVIS_KUBE_VERSION
  kubectl config set-credentials myself --username=admin --password=admin
  kubectl config set-context local --cluster=local --user=myself
  kubectl config set-cluster local --server=http://localhost:8080
  kubectl config use-context local

  # start kubernetes in the background
  sudo PATH=$PATH:/home/travis/.gimme/versions/go1.7.linux.amd64/bin/go \
       KUBE_ENABLE_CLUSTER_DNS=true \
       hack/local-up-cluster.sh &
popd

# Wait until kube is up and running
TIMEOUT=0
TIMEOUT_COUNT=800
until $(curl --output /dev/null --silent http://localhost:8080) || [ $TIMEOUT -eq $TIMEOUT_COUNT ]; do
  echo "Kube is not up yet"
  let TIMEOUT=TIMEOUT+1
  sleep 1
done

if [ $TIMEOUT -eq $TIMEOUT_COUNT ]; then
  echo "Kubernetes is not up and running"
  exit 1
fi

echo "Kubernetes is deployed and reachable"

# Try and sleep before issuing chown. Currently, Kubernetes is started by
# a command that is run in the background. Technically Kubernetes could be
# up and running, but those files might not exist yet as the previous command
# could create them after Kube starts successfully.
sleep 30
sudo chown -R $USER:$USER $HOME/.kube
