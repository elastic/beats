BEAT_TYPE=metricbeat
PREPARE_COMMAND=MODULE=elastic METRICSET=test make create-metricset ;

include ../common/Makefile

prepare-test:: python-env

	mkdir -p ${BEAT_PATH}/scripts
	rsync -a --exclude=build ${PWD}/../../metricbeat/scripts/generate_imports_helper.py ${BEAT_PATH}/scripts
