#!/usr/bin/env bash
set -exuo pipefail

MSG="parameter missing."
K8S_VERSION=${K8S_VERSION:?$MSG}
MINIKUBE_VERSION=${MINIKUBE_VERSION:?$MSG}
HOME=${HOME:?$MSG}

KBC_CMD="${HOME}/bin/kubectl"
MKB_CMD="${HOME}/bin/minikube"

export CHANGE_MINIKUBE_NONE_USER=true

mkdir -p "${HOME}/bin"

curl -sSLo "${KBC_CMD}" "https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl"
chmod +x "${KBC_CMD}"

curl -sSLo "${MKB_CMD}" "https://storage.googleapis.com/minikube/releases/${MINIKUBE_VERSION}/minikube-linux-amd64"
chmod +x "${MKB_CMD}"

mkdir -p "${HOME}/.kube" "${HOME}/.minikube"
touch "${HOME}/.kube/config"

minikube start --vm-driver=none --kubernetes-version=${K8S_VERSION} --logtostderr
minikube update-context

JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'
until kubectl get nodes -o jsonpath="${JSONPATH}" 2>&1 | grep -q "Ready=True"
do
  echo "waiting for Minikube..."
  sleep 5
done
