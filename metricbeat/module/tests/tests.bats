#!/usr/bin/env bats

load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'
load test_helpers

IMAGE="docker.elastic.co/observability-ci/${MODULE//\//-}"
CONTAINER="${MODULE//\//-}"
MODULE_UPPER=$(echo ${MODULE} | tr a-z A-Z)

@test "${MODULE} - build images" {
	cd $BATS_TEST_DIRNAME/..
	# Simplify the makefile as it does fail with '/bin/sh: 1: Bad substitution' in the CI
	if [ ! -e ${DOCKERFILE} ] ; then
		DOCKERFILE="${DOCKERFILE//-//}"
	fi

	count=$(read_versions "${DOCKERFILE}/docker-versions.yml" |wc -l)
	for (( i=0; i < $count; i++ ))
	do
		v=$(read_version "${DOCKERFILE}/docker-versions.yml" $i)

		run docker build --build-arg ${MODULE_UPPER}_VERSION=${v} --rm -t ${IMAGE}:${v} ${DOCKERFILE}
		assert_success
	done
}

@test "${MODULE} - clean test containers" {
	cleanup $CONTAINER
}

@test "${MODULE} - create test containers" {
	count=$(read_versions "${DOCKERFILE}/docker-versions.yml" |wc -l)
	for (( i=0; i < $count; i++ ))
	do
		v=$(read_version "${DOCKERFILE}/docker-versions.yml" $i)
		container=${CONTAINER}-${v}

		run docker run -d --name $container -P ${IMAGE}:${v} ${CMD}
		assert_success

	done
	assert_success
}

@test "${DOCKERFILE} - test container with 0 as exitcode" {
	count=$(read_versions "${DOCKERFILE}/docker-versions.yml" |wc -l)
	for (( i=0; i < $count; i++ ))
	do
		v=$(read_version "${DOCKERFILE}/docker-versions.yml" $i)
		container=${CONTAINER}-${v}

		sleep 1
		run docker inspect -f {{.State.ExitCode}} $container
		assert_output '0'
	done
}

@test "${MODULE} - clean test containers afterwards" {
	count=$(read_versions "${DOCKERFILE}/docker-versions.yml" |wc -l)
	for (( i=0; i < $count; i++ ))
	do
		v=$(read_version "${DOCKERFILE}/docker-versions.yml" $i)
		container=${CONTAINER}-${v}

		cleanup $container
	done
}
