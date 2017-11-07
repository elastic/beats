BEAT_TYPE=metricbeat
PREPARE_COMMAND=MODULE=elastic METRICSET=test make create-metricset ;

include ../common/Makefile

prepare-test: python-env
	mkdir -p ${BUILD_DIR}/src/github.com/elastic/beats/
	rsync -a --exclude=build ${PWD}/../../* ${BUILD_DIR}/src/github.com/elastic/beats/

	mkdir -p ${BEAT_PATH}/scripts
	rsync -a --exclude=build ${PWD}/../../metricbeat/scripts/generate_imports_helper.py ${BEAT_PATH}/scripts

	export GOPATH=${PWD}/build ; \
	. ${PYTHON_ENV}/bin/activate && python ${PWD}/build/src/github.com/elastic/beats/script/generate.py --type=${BEAT_TYPE} --project_name=Testbeat --github_name=ruflin --beat_path=beatpath/testbeat --full_name="Nicolas Ruflin"
