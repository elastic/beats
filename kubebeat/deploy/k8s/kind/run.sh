#!/usr/local/bin/bash

echo "Start" 
cd /Users/daveyakushimiso/git_repos/elastic/forks/build-sec/beats/kubebeat;
GOOS=linux go build -v
docker build -t kubebeat . 
kind load docker-image kubebeat:latest --name single-host
kubectl delete -f deploy/k8s/kubebeat-standalone-ds-local.yaml
kubectl apply -f deploy/k8s/kubebeat-standalone-ds-local.yaml
kubectl logs -f --selector="k8s-app=kubebeat" -n kube-system