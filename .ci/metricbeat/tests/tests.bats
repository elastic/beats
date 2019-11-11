#!/usr/bin/env bats

load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'
load test_helpers

# Simplify the makefile as it does fail with '/bin/sh: 1: Bad substitution' in the CI
if [ ! -e ${MODULE} ] ; then
	MODULE="${MODULE//-//}"
fi

IMAGE="docker.elastic.co/observability-ci/${MODULE//\//-}"
CONTAINER="${MODULE//\//-}"
MODULE_UPPER=$(echo ${MODULE} | tr a-z A-Z)

# Location of the Dockerfiles for each module
CONTEXT_DIR="${BATS_TEST_DIRNAME}/../../../metricbeat/module/${MODULE}/_meta"
cd ${CONTEXT_DIR}
COUNT=$(read_versions "docker-versions.yml" |wc -l)

@test "${MODULE} - build images" {
	for (( i=0; i < $COUNT; i++ ))
	do
		v=$(read_version "docker-versions.yml" $i)

		run docker build --build-arg ${MODULE_UPPER}_VERSION=${v} --rm -t ${IMAGE}:${v} .
		assert_success
	done
}

@test "${MODULE} - clean test containers" {
	cleanup $CONTAINER
}

@test "${MODULE} - create test containers" {
	for (( i=0; i < $COUNT; i++ ))
	do
		v=$(read_version "docker-versions.yml" $i)
		container=${CONTAINER}-${v}

		run docker run -d --name $container -P ${IMAGE}:${v} ${CMD}
		assert_success

	done
	assert_success
}

@test "${MODULE} - test container with 0 as exitcode" {
	for (( i=0; i < $COUNT; i++ ))
	do
		v=$(read_version "docker-versions.yml" $i)
		container=${CONTAINER}-${v}

		sleep 1
		run docker inspect -f {{.State.ExitCode}} $container
		assert_output '0'
	done
}

@test "${MODULE} - clean test containers afterwards" {
	for (( i=0; i < $COUNT; i++ ))
	do
		v=$(read_version "docker-versions.yml" $i)
		container=${CONTAINER}-${v}

		cleanup $container
	done
}
