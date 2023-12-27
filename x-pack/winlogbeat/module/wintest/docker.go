// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package wintest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// Docker starts docker-compose and waits for the services to be healthy. It returns
// a clean-up function that will conditionally stop the services, and log the docker-compose
// output to the directory specified by root, with filename TEST-elasticsearch-<target>.log.
// If verbose is true, stderr from docker-compose is passed to the test process' stderr.
// Docker is aware of STACK_ENVIRONMENT, DOCKER_NOCACHE and DOCKER_PULL.
func Docker(root, target string, verbose bool) (done func(stop bool) error, env map[string]string, _ error) {
	esBeatsDir, err := devtools.ElasticBeatsDir()
	if err != nil {
		return nil, nil, err
	}
	env = map[string]string{
		"ES_BEATS":          esBeatsDir,
		"STACK_ENVIRONMENT": devtools.StackEnvironment,
	}

	err = os.WriteFile(composeFile, []byte(compose), 0o644)
	if err != nil {
		return nil, nil, err
	}

	err = dockerCompose(env, verbose)
	if err != nil {
		return nil, nil, err
	}

	err = devtools.StartIntegTestContainers()
	if err != nil {
		return nil, nil, fmt.Errorf("starting containers: %w", err)
	}

	return func(stop bool) error {
		defer os.Remove(composeFile)

		err = saveLogs(env, root, target)
		if err != nil {
			fmt.Fprintf(os.Stdout, "failed to save docker-compose logs: %s\n", err)
		}
		if !stop {
			return nil
		}
		return devtools.StopIntegTestContainers()
	}, env, nil
}

func saveLogs(env map[string]string, root, target string) error {
	dir := filepath.Join(root, "build")
	logFile := filepath.Join(dir, "TEST-elasticsearch-"+target+".log")
	err := os.MkdirAll(dir, os.ModeDir|0o770)
	if err != nil {
		return fmt.Errorf("creating docker log dir: %w", err)
	}

	f, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("creating docker log file: %w", err)
	}
	defer f.Close()

	_, err = sh.Exec(
		env,
		f, // stdout
		f, // stderr
		"docker-compose",
		"-p", devtools.DockerComposeProjectName(),
		"logs",
		"--no-color",
	)
	if err != nil {
		return fmt.Errorf("executing docker-compose logs: %w", err)
	}
	return nil
}

const (
	composeFile = "docker-compose.yaml"
	compose     = `version: '2.3'
services:
  # This is a proxy used to block beats until all services are healthy.
  # See: https://github.com/docker/compose/issues/4369
  proxy_dep:
    image: busybox
    depends_on:
      elasticsearch: { condition: service_healthy }

  elasticsearch:
    extends:
      file: ${ES_BEATS}/testing/environments/${STACK_ENVIRONMENT}.yml
      service: elasticsearch
    healthcheck:
      test: ["CMD-SHELL", "curl -u admin:testing -s http://localhost:9200/_cat/health?h=status | grep -q green"]
      retries: 300
      interval: 1s
    ports:
      - 9200:9200
`
)

// dockerCompose runs docker-compose with the provided environment.
// It is aware of DOCKER_NOCACHE and DOCKER_PULL. If verbose is true
// the stderr output of docker-compose is written to the terminal.
func dockerCompose(env map[string]string, verbose bool) error {
	args := []string{
		"-p", devtools.DockerComposeProjectName(),
		"build",
		"--force-rm",
	}
	if _, noCache := os.LookupEnv("DOCKER_NOCACHE"); noCache {
		args = append(args, "--no-cache")
	}
	if _, forcePull := os.LookupEnv("DOCKER_PULL"); forcePull {
		args = append(args, "--pull")
	}

	out := io.Discard
	if verbose {
		out = os.Stderr
	}
	var err error
	const retries = 2
	for n := 0; n < retries; n++ {
		_, err = sh.Exec(
			env,
			out,
			os.Stderr,
			"docker-compose", args...,
		)
		if err == nil {
			break
		}
		// This sleep is to avoid hitting the docker build
		// issues when resources are not available.
		time.Sleep(10 * time.Nanosecond)
	}
	return err
}
