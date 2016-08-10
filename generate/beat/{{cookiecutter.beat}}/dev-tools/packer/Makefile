BUILDID=$(shell git rev-parse HEAD)

.PHONY: all
all: {{cookiecutter.beat}}/deb \
     {{cookiecutter.beat}}/rpm \
	 {{cookiecutter.beat}}/darwin \
	 {{cookiecutter.beat}}/win \
	 {{cookiecutter.beat}}/bin \
	 build/upload/build_id.txt

# The generator requires the dependencies to be in the vendor/ folder
ES_BEATS=../../vendor/github.com/elastic/beats
include $(ES_BEATS)/dev-tools/packer/scripts/Makefile


.PHONY: {{cookiecutter.beat}}
{{cookiecutter.beat}}: build/upload
	# cross compile on ubuntu
	docker run --rm \
		-v $(abspath build):/build \
		-v $(abspath $(ES_BEATS)/dev-tools/packer/xgo-scripts):/scripts \
		-v $(abspath ../..):/source \
		-e PUREGO="yes" \
		-e PACK=$@ \
		-e BEFORE_BUILD=before_build.sh \
		-e SOURCE=/source \
		-e TARGETS="linux/amd64 linux/386 windows/amd64 windows/386 darwin/amd64" \
		-e BUILDID=${BUILDID} \
		tudorg/beats-builder \
		{{cookiecutter.beat_path}}/{{cookiecutter.beat}}
