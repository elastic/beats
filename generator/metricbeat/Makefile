BEAT_TYPE=metricbeat
PREPARE_COMMAND=MODULE=elastic METRICSET=test make create-metricset ; FIELDS_FILE_PATH=module
override FIELDS_FILE_PATH=
export FIELDS_FILE_PATH


include ../common/Makefile

prepare-test:: python-env

	mkdir -p ${BEAT_PATH}/scripts
	rsync -a --exclude=build ${PWD}/../../metricbeat/scripts/generate_imports_helper.py ${BEAT_PATH}/scripts

# Collects all dependencies and then calls update
.PHONY: collect
collect: fields
