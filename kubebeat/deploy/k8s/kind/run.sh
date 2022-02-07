#!/usr/local/bin/bash

echo "Start" 
cd /Users/daveyakushimiso/git_repos/elastic/forks/build-sec/beats/cloudbeat;
GOOS=linux go build -v
docker build -t cloudbeat . 
kind load docker-image cloudbeat:latest --name single-host
kubectl delete -f deploy/k8s/cloudbeat-standalone-ds-local.yaml
kubectl apply -f deploy/k8s/cloudbeat-standalone-ds-local.yaml
kubectl logs -f --selector="k8s-app=cloudbeat" -n kube-system