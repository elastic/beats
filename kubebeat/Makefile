BEAT_NAME=kubebeat
BEAT_PATH=github.com/elastic/beats/v7/kubebeat
BEAT_GOPATH=$(firstword $(subst :, ,${GOPATH}))
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false
ES_BEATS_IMPORT_PATH=github.com/elastic/beats/v7
ES_BEATS?=$(shell go list -m -f '{{.Dir}}' ${ES_BEATS_IMPORT_PATH})
LIBBEAT_MAKEFILE=$(ES_BEATS)/libbeat/scripts/Makefile
GOPACKAGES=$(shell go list ${BEAT_PATH}/... | grep -v /tools)
GOBUILD_FLAGS=-i -ldflags "-X ${ES_BEATS_IMPORT_PATH}/libbeat/version.buildTime=$(NOW) -X ${ES_BEATS_IMPORT_PATH}/libbeat/version.commit=$(COMMIT_ID)"
MAGE_IMPORT_PATH=github.com/magefile/mage
NO_COLLECT=true
CHECK_HEADERS_DISABLED=true

# Path to the libbeat Makefile
-include $(LIBBEAT_MAKEFILE)

.PHONY: copy-vendor
copy-vendor:
	mage vendorUpdate

delete-pod:
	kubectl delete pod kubebeat-demo

build-docker:
	GOOS=linux go build && docker build -t kubebeat .

docker-image-load-minikube: build-docker
	minikube image load kubebeat:latest

docker-image-load-kind: build-docker
	kind load docker-image docker.elastic.co/beats/elastic-agent:8.1.0-SNAPSHOT --name single-host

deploy-kubebeat:
	kubectl apply -f deploy/k8s/kubebeat-ds.yaml -n kube-system

deploy-pod: delete-pod build-docker docker-image-load-minikube
	kubectl apply -f pod.yml

build-deploy-docker: build-docker docker-image-load-kind deploy-kubebeat

