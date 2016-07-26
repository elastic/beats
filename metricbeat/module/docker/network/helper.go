package network

import "github.com/elastic/beats/metricbeat/module/docker/vendor/github.com/fsouza/go-dockerclient"

type NETService struct {}

type NETRaw struct {
	Time      time.Time
	RxBytes   uint64
	RxDropped uint64
	RxErrors  uint64
	RxPackets uint64
	TxBytes   uint64
	TxDropped uint64
	TxErrors  uint64
	TxPackets uint64
}

func getNewNet(stats docker.Stats) NETRaw{


}
func getOldNet(stats docker.Stats) NETRaw{

}
