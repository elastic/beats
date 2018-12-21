# This script assumes Docker is already installed
#!/bin/bash

set -x
set -e

export CHANGE_MINIKUBE_NONE_USER=true

mkdir -p $HOME/bin/
curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.10.0/bin/linux/amd64/kubectl && \
      chmod +x kubectl && mv kubectl $HOME/bin/
curl -Lo minikube https://storage.googleapis.com/minikube/releases/v0.25.2/minikube-linux-amd64 && chmod +x minikube && mv minikube $HOME/bin/
$HOME/bin/minikube start --vm-driver=none --kubernetes-version=v1.10.0 --logtostderr
$HOME/bin/minikube update-context
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; \
        until $HOME/bin/kubectl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done

