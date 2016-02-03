BEATNAME={{cookiecutter.beat}}
BEAT_DIR={{cookiecutter.beat_path}}
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false
ES_BEATS=./vendor/github.com/elastic/beats
GOPACKAGES=$(shell glide novendor)
PREFIX?=.

# Path to the libbeat Makefile
include $(ES_BEATS)/libbeat/scripts/Makefile


.PHONY: generate
generate:
	python scripts/generate_template.py    etc/fields.yml    etc/{{cookiecutter.beat}}.template.json
	python scripts/generate_field_docs.py  etc/fields.yml    etc/{{cookiecutter.beat}}.asciidoc

.PHONY: install-cfg
install-cfg:
	mkdir -p $(PREFIX)
	cp etc/{{cookiecutter.beat}}.template.json     $(PREFIX)/{{cookiecutter.beat}}.template.json
	cp etc/{{cookiecutter.beat}}.yml               $(PREFIX)/{{cookiecutter.beat}}.yml
	cp etc/{{cookiecutter.beat}}.yml               $(PREFIX)/{{cookiecutter.beat}}-linux.yml
	cp etc/{{cookiecutter.beat}}.yml               $(PREFIX)/{{cookiecutter.beat}}-binary.yml
	cp etc/{{cookiecutter.beat}}.yml               $(PREFIX)/{{cookiecutter.beat}}-darwin.yml
	cp etc/{{cookiecutter.beat}}.yml               $(PREFIX)/{{cookiecutter.beat}}-win.yml

.PHONY: update-deps
update-deps:
	glide update  --no-recursive
