default: test

export PATH := $(CURDIR)/$(VENV)/bin:$(PATH)

VENV ?= .venv
PYTHON ?= python3
PIP ?= pip3

SHELL = /bin/bash
MAKEFLAGS += --silent --no-print-directory
.SHELLFLAGS = -ec
.SILENT:

.ONESHELL:

.PHONY: help
help:
	@grep '^## @help' Makefile|cut -d ":" -f 2-3|( (sort|column -s ":" -t) || (sort|tr ":" "\t") || (tr ":" "\t"))

## @help:virtualenv:Create a Python virtual environment.
.PHONY: virtualenv
virtualenv:
	$(PYTHON) --version
	test -d $(VENV) || virtualenv -q --python=$(PYTHON) $(VENV);\
	source $(VENV)/bin/activate;\
	$(PIP) install -r requirements.txt;

## @help:deps:Update helm charts dependencies
.PHONY: deps
deps:
	helm dependency update

## @help:lint:Lint helm templates.
.PHONY: lint
lint:
	helm lint --strict ./

## @help:lint-python:Lint Python scripts.
.PHONY: lint-python
lint-python: virtualenv
	source $(VENV)/bin/activate;\
	black --diff --check --exclude='$(VENV)' .

## @help:pytest:Run python tests.
.PHONY: pytest
pytest: virtualenv
	source $(VENV)/bin/activate;\
	pytest -sv --color=yes

## @help:template:Render chart templates.
.PHONY: template
template:
	helm template ./

## @help:test-all:Run all tests.
.PHONY: test-all
test-all: lint deps template pytest

## @help:clean:Clean the temporal files.
.PHONY: clean
clean:
	rm -fr $(VENV)
	rm -fr .pytest_cache
