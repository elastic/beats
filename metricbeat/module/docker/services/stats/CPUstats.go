package stats

import (
	"github.com/elastic/beats/libbeat/logp"
	"github.com/fsouza/go-dockerclient"
	"fmt"
	"sync"
	"github.com/elastic/beats/metricbeat/module/docker/services/config"
)
func GetCPUStats() ([]config.CPUData) {

	InitDockerClient()
	logp.Info(" DockerStat is running")
	myStats, err:= FetchStats()
	if err == nil {
		if len(myStats) != 0{
			logp.Info(" Great, stats are available! \n")
		} else{
			logp.Info(" No container is running! \n")
		}
	}else {
		logp.Info(" Impossible to get stats \n")
	}
	return myStats
}

func FetchStats() ([]config.CPUData, error ){
	containers, err := myDockerstat.dockerClient.ListContainers(docker.ListContainersOptions{})

	myEvents := []config.CPUData{}
	if err == nil {
		//export stats for each container
		for _, container := range containers {
			myEvents = append(myEvents, ExportContainerStats(container))
		}
	} else {
		logp.Err("Can not get container list: %v", err)
	}
	fmt.Printf(" FetchSTats taille : ", len(myEvents))
	return myEvents, err
}
func  ExportContainerStats(container docker.APIContainers) config.CPUData {
	// statsOptions creation
	var wg sync.WaitGroup
	statsC := make(chan *docker.Stats)
	errC := make(chan error, 1)

	events := config.CPUData{}
	// the stream bool is set to false to only listen the first stats
	statsOptions := docker.StatsOptions{
		ID:      container.ID,
		Stats:   statsC,
		Stream:  false,
		Timeout: -1,
	}
	wg.Add(2)
	// goroutine to listen to the stats
	go func() {
		defer wg.Done()
		errC <- myDockerstat.dockerClient.Stats(statsOptions)
		close(errC)
	}()
	// goroutine to get the stats & publish it
	go func() {
		defer wg.Done()
		stats := <-statsC
		err := <-errC

		if err == nil && stats != nil {
			events = myDockerstat.dataGenerator.GetCpuData(&container,stats)
		} else if err == nil && stats == nil {
			logp.Warn("Container was existing at listing but not when getting statistics: %v", container.ID)

		} else {
			logp.Err("An error occurred while getting docker stats: %v", err)

		}
	}()
	wg.Wait()
	return events
}
