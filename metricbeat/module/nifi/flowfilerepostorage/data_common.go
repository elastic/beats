package flowfilerepostorage

// FlowFileRepositoryStorageUsage ...
type FlowFileRepositoryStorageUsage struct {
	FreeSpace       string `json:"freeSpace"`
	FreeSpaceBytes  uint64 `json:"freeSpaceBytes"`
	TotalSpace      string `json:"totalSpace"`
	TotalSpaceBytes uint64 `json:"totalSpaceBytes"`
	UsedSpace       string `json:"usedSpace"`
	UsedSpaceBytes  uint64 `json:"usedSpaceBytes"`
	Utilization     string `json:"utilization"`
}
