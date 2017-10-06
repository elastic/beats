ALL = filebeat metricbeat
BEAT_VERSION=$(shell head -n 1 ../../libbeat/docs/version.asciidoc | cut -c 17- )

all: ${ALL:=-kubernetes.yaml}

test: all
	ls *-kubernetes-${BEAT_VERSION}.yaml
	for FILE in $(shell ls *-kubernetes-${BEAT_VERSION}.yaml); do \
		BEAT=$$(echo $$FILE | cut -d \- -f 1); \
		kubectl create -f $$FILE; \
	done

clean:
	@for f in $(ALL); do rm -f "$$f-kubernetes-${BEAT_VERSION}.yaml"; done

%-kubernetes.yaml: %/*.yaml
	@echo "Generating $*-kubernetes-${BEAT_VERSION}.yaml"
	@rm -f $*-kubernetes-${BEAT_VERSION}.yaml
	@for f in $(shell ls $*/*.yaml); do \
		sed "s/%VERSION%/${BEAT_VERSION}/g" $$f >> $*-kubernetes-${BEAT_VERSION}.yaml; \
		echo --- >> $*-kubernetes-${BEAT_VERSION}.yaml; \
	done
