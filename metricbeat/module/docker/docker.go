package docker

import (
	"sync"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"

	"github.com/fsouza/go-dockerclient"
)

var HostParser = parse.URLHostParserBuilder{DefaultScheme: "tcp"}.Build()

func init() {
	// Register the ModuleFactory function for the "docker" module.
	if err := mb.Registry.AddModule("docker", NewModule); err != nil {
		panic(err)
	}
}

func NewModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := struct {
		Hosts []string `config:"hosts"    validate:"nonzero,required"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &base, nil
}

type Stat struct {
	Container docker.APIContainers
	Stats     docker.Stats
}

func NewDockerClient(endpoint string, config Config) (*docker.Client, error) {
	var err error
	var client *docker.Client

	if !config.TLS.IsEnabled() {
		client, err = docker.NewClient(endpoint)
	} else {
		client, err = docker.NewTLSClient(
			endpoint,
			config.TLS.Certificate,
			config.TLS.Key,
			config.TLS.CA,
		)
	}
	if err != nil {
		return nil, err
	}

	return client, nil
}

// FetchStats returns a list of running containers with all related stats inside
func FetchStats(client *docker.Client) ([]Stat, error) {
	containers, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup

	containersList := []Stat{}
	queue := make(chan Stat, 1)
	wg.Add(len(containers))

	for _, container := range containers {
		go func(container docker.APIContainers) {
			queue <- exportContainerStats(client, &container)
		}(container)
	}

	go func() {
		for container := range queue {
			containersList = append(containersList, container)
			wg.Done()
		}
	}()

	wg.Wait()

	return containersList, err
}

// exportContainerStats loads stats for the given container
//
// This is currently very inefficient as docker calculates the average for each request,
// means each request will take at least 2s: https://github.com/docker/docker/blob/master/cli/command/container/stats_helpers.go#L148
// Getting all stats at once is implemented here: https://github.com/docker/docker/pull/25361
func exportContainerStats(client *docker.Client, container *docker.APIContainers) Stat {
	var wg sync.WaitGroup
	var event Stat

	statsC := make(chan *docker.Stats)
	errC := make(chan error, 1)
	statsOptions := docker.StatsOptions{
		ID:      container.ID,
		Stats:   statsC,
		Stream:  false,
		Timeout: -1,
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		errC <- client.Stats(statsOptions)
		close(errC)
	}()
	go func() {
		defer wg.Done()
		stats := <-statsC
		err := <-errC
		if stats != nil && err == nil {
			event.Stats = *stats
			event.Container = *container
		} else if err == nil && stats == nil {
			logp.Warn("Container stopped when recovering stats: %v", container.ID)
		} else {
			logp.Err("An error occurred while getting docker stats: %v", err)
		}
	}()
	wg.Wait()
	return event
}
