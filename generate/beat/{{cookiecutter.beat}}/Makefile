BEATNAME={{cookiecutter.beat}}
BEAT_DIR={{cookiecutter.beat_path}}
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false
ES_BEATS?=${GOPATH}/src/github.com/elastic/beats
GOPACKAGES=$(shell glide novendor)
PREFIX?=.

# Path to the libbeat Makefile
-include $(ES_BEATS)/libbeat/scripts/Makefile

.PHONY: init
init:
	glide update --no-recursive
	make update
	git init

.PHONY: commit
commit:
	git add README.md CONTRIBUTING.md
	git commit -m "Initial commit"
	git add LICENSE
	git commit -m "Add the LICENSE"
	git add .gitignore
	git commit -m "Add git settings"
	git add .
	git reset -- .travis.yml
	git commit -m "Add {{cookiecutter.beat}}"
	git add .travis.yml
	git commit -m "Add Travis CI"

.PHONY: update-deps
update-deps:
	glide update --no-recursive --strip-vcs

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:

# Create binary packages for the beat
pack: create-packer
	cd dev-tools/packer; make deps images {{cookiecutter.beat}}
