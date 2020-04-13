# This script assumes Docker is already installed
#!/bin/bash

set -x
set -e

export CHANGE_MINIKUBE_NONE_USER=true

curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/$TRAVIS_K8S_VERSION/bin/linux/amd64/kubectl && \
      chmod +x kubectl && sudo mv kubectl /usr/local/bin/
curl -Lo minikube https://storage.googleapis.com/minikube/releases/$TRAVIS_MINIKUBE_VERSION/minikube-linux-amd64 && chmod +x minikube && sudo mv minikube /usr/local/bin/
mkdir -p $HOME/.kube $HOME/.minikube
touch $HOME/.kube/config
sudo minikube start --vm-driver=none --kubernetes-version=$TRAVIS_K8S_VERSION --logtostderr
sudo minikube update-context
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; \
        until kubectl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done

