package system

type Response struct {
	Status string `xml:"status,attr"`
	Result string `xml:"result"`
}

type SystemLoad struct {
	one_minute     float64
	five_minute    float64
	fifteen_minute float64
}

type Uptime struct {
	Days  int
	Hours string
}

type SystemInfo struct {
	Uptime      Uptime
	UserCount   int
	LoadAverage SystemLoad
}

type TaskInfo struct {
	Total    int
	Running  int
	Sleeping int
	Stopped  int
	Zombie   int
}

type CPUInfo struct {
	User      float64
	System    float64
	Nice      float64
	Idle      float64
	Wait      float64
	Hi        float64
	SystemInt float64
	Steal     float64
}

type MemoryInfo struct {
	Total       float64
	Free        float64
	Used        float64
	BufferCache float64
}

type SwapInfo struct {
	Total     float64
	Free      float64
	Used      float64
	Available float64
}
