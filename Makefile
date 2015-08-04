
.PHONY: image
image:
	docker build -t tudorg/beats-builder xgo-image/

.PHONY: packetbeat
packetbeat: image
	cd build && xgo -image=tudorg/beats-builder -static \
		-before-build=../xgo-scripts/before_build.sh \
		github.com/elastic/packetbeat

.PHONY: run-interactive
run-interactive:
	docker run -t -i -v $(shell pwd):/build -v $(shell pwd)/xgo-scripts/:/scripts --entrypoint=bash tudorg/beats-builder

