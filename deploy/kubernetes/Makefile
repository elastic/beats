ALL=filebeat metricbeat auditbeat heartbeat
IMAGE_MODIFIER?="-wolfi"
BEAT_VERSION=$(shell head -n 1 ../../libbeat/docs/version.asciidoc | cut -c 17- )

.PHONY: all $(ALL)

all: $(ALL)

test: all
	@for BEAT in $(ALL); do \
		echo; \
		echo "$$BEAT"; \
		FILE="$$BEAT-kubernetes.yaml"; \
		kubectl create -f $$FILE; \
		echo "Testing $$BEAT container for readiness..."; \
		kubectl wait pods -n kube-system -l k8s-app=$$BEAT --for=condition=Ready --timeout=90s; \
		echo "Deleting $$BEAT..."; \
		kubectl delete -f $$FILE; \
	done

clean:
	@for f in $(ALL); do rm -f "$$f-kubernetes.yaml"; done

$(ALL):
	@echo "Generating $@-kubernetes.yaml for version ${BEAT_VERSION} and image modifier '${IMAGE_MODIFIER}'"
	@rm -f $@-kubernetes.yaml
	@for f in service-account role role-binding configmap deployment daemonset ; do \
		if [ -f "$@/$@-$$f.yaml" ]; then \
			echo "file: $@/$@-$$f.yaml"; \
			cat $@/$@-$$f.yaml | sed "s/%VERSION%/${BEAT_VERSION}/g" | sed "s/%IMAGE_MODIFIER%/${IMAGE_MODIFIER}/g" >> $@-kubernetes.yaml; \
			echo --- >> $@-kubernetes.yaml; \
		fi \
	done
