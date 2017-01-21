package disk

import (
	"encoding/json"
)

type UsageStat struct {
	Path              string  `json:"path"`
	Fstype            string  `json:"fstype"`
	Total             uint64  `json:"total"`
	Free              uint64  `json:"free"`
	Used              uint64  `json:"used"`
	UsedPercent       float64 `json:"usedPercent"`
	InodesTotal       uint64  `json:"inodesTotal"`
	InodesUsed        uint64  `json:"inodesUsed"`
	InodesFree        uint64  `json:"inodesFree"`
	InodesUsedPercent float64 `json:"inodesUsedPercent"`
}

type PartitionStat struct {
	Device     string `json:"device"`
	Mountpoint string `json:"mountpoint"`
	Fstype     string `json:"fstype"`
	Opts       string `json:"opts"`
}

type IOCountersStat struct {
	ReadCount    uint64 `json:"readCount"`
	WriteCount   uint64 `json:"writeCount"`
	ReadBytes    uint64 `json:"readBytes"`
	WriteBytes   uint64 `json:"writeBytes"`
	ReadTime     uint64 `json:"readTime"`
	WriteTime    uint64 `json:"writeTime"`
	Name         string `json:"name"`
	IoTime       uint64 `json:"ioTime"`
	SerialNumber string `json:"serialNumber"`
}

func (d UsageStat) String() string {
	s, _ := json.Marshal(d)
	return string(s)
}

func (d PartitionStat) String() string {
	s, _ := json.Marshal(d)
	return string(s)
}

func (d IOCountersStat) String() string {
	s, _ := json.Marshal(d)
	return string(s)
}
