package heap

// Heap ...
type Heap struct {
	TotalNonHeap      string `json:"totalNonHeap"`
	TotalNonHeapBytes int64  `json:"totalNonHeapBytes"`
	UsedNonHeap       string `json:"usedNonHeap"`
	UsedNonHeapBytes  int64  `json:"usedNonHeapBytes"`
	FreeNonHeap       string `json:"freeNonHeap"`
	FreeNonHeapBytes  int64  `json:"freeNonHeapBytes"`
	MaxNonHeap        string `json:"maxNonHeap"`
	MaxNonHeapBytes   int64  `json:"maxNonHeapBytes"`
	TotalHeap         string `json:"totalHeap"`
	TotalHeapBytes    int64  `json:"totalHeapBytes"`
	UsedHeap          string `json:"usedHeap"`
	UsedHeapBytes     int64  `json:"usedHeapBytes"`
	FreeHeap          string `json:"freeHeap"`
	FreeHeapBytes     int64  `json:"freeHeapBytes"`
	MaxHeap           string `json:"maxHeap"`
	MaxHeapBytes      int64  `json:"maxHeapBytes"`
	HeapUtilization   string `json:"heapUtilization"`
}
