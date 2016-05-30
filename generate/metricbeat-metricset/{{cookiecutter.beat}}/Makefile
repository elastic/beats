BEATNAME={{cookiecutter.beat}}
BEAT_DIR={{cookiecutter.beat_path}}
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false
ES_BEATS=./vendor/github.com/elastic/beats
GOPACKAGES=$(shell glide novendor)
PREFIX?=.

# Path to the libbeat Makefile
-include $(ES_BEATS)/metricbeat/Makefile

.PHONY: init
init:
	glide --debug update  --no-recursive --use-gopath
	make collect
	git init


.PHONY: commit
commit:
	git add README.md CONTRIBUTING.md
	git commit -m "Initial commit"
	git add LICENSE
	git commit -m "Add the LICENSE"
	git add .gitignore .gitattributes
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
