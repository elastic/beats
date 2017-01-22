package flowfilerepostorage

// FlowFileRepositoryStorageUsage ...
type FlowFileRepositoryStorageUsage struct {
	FreeSpace       string `json:"freeSpace"`
	FreeSpaceBytes  int64  `json:"freeSpaceBytes"`
	TotalSpace      string `json:"totalSpace"`
	TotalSpaceBytes int64  `json:"totalSpaceBytes"`
	UsedSpace       string `json:"usedSpace"`
	UsedSpaceBytes  int64  `json:"usedSpaceBytes"`
	Utilization     string `json:"utilization"`
}
