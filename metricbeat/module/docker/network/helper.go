package network

import (
	"time"

	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/metricbeat/module/docker"
)

type NETService struct {
	NetworkStatPerContainer map[string]map[string]NETRaw
}

type NetworkCalculator interface {
	getRxBytesPerSecond(newStats *NETRaw, oldStats *NETRaw) float64
	getRxDroppedPerSecond(newStats *NETRaw, oldStats *NETRaw) float64
	getRxErrorsPerSecond(newStats *NETRaw, oldStats *NETRaw) float64
	getRxPacketsPerSecond(newStats *NETRaw, oldStats *NETRaw) float64
	getTxBytesPerSecond(newStats *NETRaw, oldStats *NETRaw) float64
	getTxDroppedPerSecond(newStats *NETRaw, oldStats *NETRaw) float64
	getTxErrorsPerSecond(newStats *NETRaw, oldStats *NETRaw) float64
	getTxPacketsPerSecond(newStats *NETRaw, oldStats *NETRaw) float64
}

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

type NETstats struct {
	Time          time.Time
	Container     *docker.Container
	NameInterface string
	RxBytes       float64
	RxDropped     float64
	RxErrors      float64
	RxPackets     float64
	TxBytes       float64
	TxDropped     float64
	TxErrors      float64
	TxPackets     float64
}

func (NT *NETService) getNetworkStatsPerContainer(rawStats []docker.DockerStat) []NETstats {
	formatedStats := []NETstats{}
	for _, myStats := range rawStats {
		for nameInterface, rawnNetStats := range myStats.Stats.Networks {
			formatedStats = append(formatedStats, NT.getNetworkStats(nameInterface, &rawnNetStats, &myStats))
		}
	}

	return formatedStats
}

func (NT *NETService) getNetworkStats(nameInterface string, rawNetStats *dc.NetworkStats, myRawstats *docker.DockerStat) NETstats {

	newNetworkStats := newNETRAw(myRawstats.Stats.Read, rawNetStats)
	oldNetworkStat, exist := NT.NetworkStatPerContainer[myRawstats.Container.ID][nameInterface]

	netStats := NETstats{
		Container:     docker.NewContainer(&myRawstats.Container),
		Time:          myRawstats.Stats.Read,
		NameInterface: nameInterface,
	}

	if exist {
		netStats.RxBytes = NT.getRxBytesPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.RxDropped = NT.getRxDroppedPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.RxErrors = NT.getRxErrorsPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.RxPackets = NT.getRxPacketsPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.TxBytes = NT.getTxBytesPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.TxDropped = NT.getTxDroppedPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.TxErrors = NT.getTxErrorsPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.TxPackets = NT.getTxPacketsPerSecond(&newNetworkStats, &oldNetworkStat)
	} else {
		NT.NetworkStatPerContainer[myRawstats.Container.ID] = make(map[string]NETRaw)
	}

	NT.NetworkStatPerContainer[myRawstats.Container.ID][nameInterface] = newNetworkStats

	return netStats

}

func newNETRAw(time time.Time, stats *dc.NetworkStats) NETRaw {
	return NETRaw{
		Time:      time,
		RxBytes:   stats.RxBytes,
		RxDropped: stats.RxDropped,
		RxErrors:  stats.RxErrors,
		RxPackets: stats.RxPackets,
		TxBytes:   stats.TxBytes,
		TxDropped: stats.TxDropped,
		TxErrors:  stats.TxErrors,
		TxPackets: stats.TxPackets,
	}

}

func (NT *NETService) checkStats(containerID string, nameInterface string) bool {
	if _, exist := NT.NetworkStatPerContainer[containerID][nameInterface]; exist {
		return true
	}
	return false
}

func (NT *NETService) getRxBytesPerSecond(newStats *NETRaw, oldStats *NETRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return NT.calculatePerSecond(duration, oldStats.RxBytes, newStats.RxBytes)
}

func (NT *NETService) getRxDroppedPerSecond(newStats *NETRaw, oldStats *NETRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return NT.calculatePerSecond(duration, oldStats.RxDropped, newStats.RxDropped)
}

func (NT *NETService) getRxErrorsPerSecond(newStats *NETRaw, oldStats *NETRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return NT.calculatePerSecond(duration, oldStats.RxErrors, newStats.RxErrors)
}

func (NT *NETService) getRxPacketsPerSecond(newStats *NETRaw, oldStats *NETRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return NT.calculatePerSecond(duration, oldStats.RxPackets, newStats.RxPackets)
}

func (NT *NETService) getTxBytesPerSecond(newStats *NETRaw, oldStats *NETRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return NT.calculatePerSecond(duration, oldStats.TxBytes, newStats.TxBytes)
}

func (NT *NETService) getTxDroppedPerSecond(newStats *NETRaw, oldStats *NETRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return NT.calculatePerSecond(duration, oldStats.TxDropped, newStats.TxDropped)
}

func (NT *NETService) getTxErrorsPerSecond(newStats *NETRaw, oldStats *NETRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return NT.calculatePerSecond(duration, oldStats.TxErrors, newStats.TxErrors)
}

func (NT *NETService) getTxPacketsPerSecond(newStats *NETRaw, oldStats *NETRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return NT.calculatePerSecond(duration, oldStats.TxPackets, newStats.TxPackets)
}

func (NT *NETService) calculatePerSecond(duration time.Duration, oldValue uint64, newValue uint64) float64 {
	value := float64(newValue) - float64(oldValue)
	if value < 0 {
		value = 0
	}
	return value / duration.Seconds()
}
