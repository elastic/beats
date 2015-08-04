.PHONY: deps
deps:
	go get github.com/tsg/xgo

.PHONY: image
image:
	docker build -t tudorg/beats-builder xgo-image/

build:
	mkdir -p build

.PHONY: packetbeat
packetbeat: image build
	cd build && xgo -image=tudorg/beats-builder -static \
		-before-build=../xgo-scripts/before_build.sh \
		github.com/elastic/packetbeat

.PHONY: run-interactive
run-interactive:
	docker run -t -i -v $(shell pwd)/build:/build \
		-v $(shell pwd)/xgo-scripts/:/scripts \
		--entrypoint=bash tudorg/beats-builder
.PHONY: clean
clean:
	rm -rf build/ || true
