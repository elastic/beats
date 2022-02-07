# Todo delete before merge to elastic/beats
create-kind-cluster:
  kind create cluster --config deploy/k8s/kind/kind-config.yaml

install-kind:
  brew install kind

setup-env: install-kind create-kind-cluster

load-cloudbeat-image:
  kind load docker-image cloudbeat:latest --name kind-mono

load-agent-image:
  kind load docker-image docker.elastic.co/beats/elastic-agent:8.1.0-SNAPSHOT --name kind-mono

build-cloudbeat:
  GOOS=linux go build -v && docker build -t cloudbeat .

deploy-cloudbeat:
  kubectl delete -f deploy/k8s/cloudbeat-ds.yaml -n kube-system & kubectl apply -f deploy/k8s/cloudbeat-ds.yaml -n kube-system

build-deploy-cloudbeat: build-cloudbeat load-cloudbeat-image deploy-cloudbeat

build-cloudbeat-debug:
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -gcflags "all=-N -l" && docker build -f Dockerfile.debug -t cloudbeat .

deploy-cloudbeat-debug:
  kubectl delete -f deploy/k8s/cloudbeat-ds-debug.yaml -n kube-system & kubectl apply -f deploy/k8s/cloudbeat-ds-debug.yaml -n kube-system

build-deploy-cloudbeat-debug: build-cloudbeat-debug load-cloudbeat-image deploy-cloudbeat-debug

logs-cloudbeat:
  kubectl logs -f --selector="k8s-app=cloudbeat" -n kube-system

package-agent:
  cd ../x-pack/elastic-agent & DEV=true SNAPSHOT=true PLATFORMS=linux/amd64 TYPES=docker mage -v package

deploy-agent:
  kubectl delete -f deploy/k8s/fleet-managed-agent.yaml -n kube-system & kubectl apply -f deploy/k8s//fleet-managed-agent.yaml -n kube-system

build-deploy-agent: package-agent load-agent-image deploy-agent

build-kibana-docker:
  node scripts/build --docker-images --skip-docker-ubi --skip-docker-centos -v

elastic-stack-up:
  elastic-package stack up --version=8.1.0-SNAPSHOT

elastic-stack-down:
  elastic-package stack down

