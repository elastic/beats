package docker

import (
	"sync"

	"github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/logp"
)

type DockerStat struct {
	Container docker.APIContainers
	Stats     docker.Stats
}

// TODO: These should not be global as otherwise only one client and socket can be used -> max 1 module to monitor
var socket string

func NewDockerClient(config *Config) (*docker.Client, error) {
	socket = config.Socket

	var err error
	var client *docker.Client

	if config.Tls.Enabled == true {
		client, err = docker.NewTLSClient(
			config.Socket,
			config.Tls.CertPath,
			config.Tls.KeyPath,
			config.Tls.CaPath,
		)
	} else {
		client, err = docker.NewClient(config.Socket)
	}
	if err != nil {
		return nil, err
	}

	logp.Info("Docker client is created")

	return client, nil
}

// FetchStats returns a list of running containers with all related stats inside
func FetchStats(client *docker.Client) ([]DockerStat, error) {
	containers, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return nil, err
	}

	containersList := []DockerStat{}
	for _, container := range containers {
		containersList = append(containersList, exportContainerStats(client, &container))
	}

	return containersList, err
}

func exportContainerStats(client *docker.Client, container *docker.APIContainers) DockerStat {
	var wg sync.WaitGroup
	var event DockerStat

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

func GetSocket() string {
	return socket
}
