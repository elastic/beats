package stats

import (
	"github.com/elastic/beats/libbeat/logp"
	"github.com/fsouza/go-dockerclient"
	"sync"
	"github.com/elastic/beats/metricbeat/module/docker/services/config"
)
func GetMEMStats() ([]config.MEMORYData) {

	InitDockerClient()
	logp.Info(" DockerStat is running")
	myStats, err:= FetchMEMStats()
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

func FetchMEMStats() ([]config.MEMORYData, error ){
	containers, err := myDockerstat.dockerClient.ListContainers(docker.ListContainersOptions{})

	myEvents := []config.MEMORYData{}
	if err == nil {
		//export stats for each container
		for _, container := range containers {
			myEvents = append(myEvents, ExportContainerMEMStats(container))
		}
	} else {
		logp.Err("Can not get container list: %v", err)
	}
	//fmt.Printf(" FetchSTats taille : ", len(myEvents))
	return myEvents, err
}
func  ExportContainerMEMStats(container docker.APIContainers) config.MEMORYData {
	// statsOptions creation
	var wg sync.WaitGroup
	statsC := make(chan *docker.Stats)
	errC := make(chan error, 1)

	events := config.MEMORYData{}
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
			events = myDockerstat.dataGenerator.GetMemoryData(&container,stats)
		} else if err == nil && stats == nil {
			logp.Warn("Container was existing at listing but not when getting statistics: %v", container.ID)

		} else {
			logp.Err("An error occurred while getting docker stats: %v", err)

		}
	}()
	wg.Wait()
	return events
}
