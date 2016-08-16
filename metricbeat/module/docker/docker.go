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

var socket string
var client *docker.Client

func CreateDockerCLient(config *Config) *docker.Client {
	socket = config.Socket
	var err error
	if client == nil {
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
		if err == nil {
			logp.Info("DockerCLient is created")
			return client
		} else {
			logp.Info("DockerCLient is not created")
		}
	} else {
		logp.Info("DockerCLient already exists")
		return client
	}
	return nil
}
func FetchDockerStats(client *docker.Client) ([]DockerStat, error) {
	containers, err := client.ListContainers(docker.ListContainersOptions{})
	containersList := []DockerStat{}
	if err == nil {
		for _, container := range containers {
			containersList = append(containersList, exportContainerStats(client, &container))
		}
	} else {
		logp.Err("Can not get container list: %v", err)
	}
	return containersList, err
}
func exportContainerStats(client *docker.Client, container *docker.APIContainers) DockerStat {
	var wg sync.WaitGroup
	statsC := make(chan *docker.Stats)
	errC := make(chan error, 1)
	var event DockerStat
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
