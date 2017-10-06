ALL = filebeat metricbeat

all: ${ALL:=-kubernetes.yaml}

test: all
	ls *-kubernetes.yaml
	for FILE in $(shell ls *-kubernetes.yaml); do \
		BEAT=$$(echo $$FILE | cut -d \- -f 1); \
		kubectl create -f $$FILE; \
	done

clean:
	@for f in $(ALL); do rm -f "$$f-kubernetes.yaml"; done

%-kubernetes.yaml: %/*.yaml
	@echo "Generating $*-kubernetes.yaml"
	@rm -f $*-kubernetes.yaml
	@for f in $(shell ls $*/*.yaml); do \
		echo --- >> $*-kubernetes.yaml; \
		cat $$f >> $*-kubernetes.yaml; \
	done
