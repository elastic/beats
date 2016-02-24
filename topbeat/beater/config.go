package beater

type TopConfig struct {
	Period *int64
	Procs  *[]string
	Stats  struct {
		System     *bool `config:"system"`
		Proc       *bool `config:"process"`
		Filesystem *bool `config:"filesystem"`
		CpuPerCore *bool `config:"cpu_per_core"`
	}
}

type ConfigSettings struct {
	Input TopConfig
}
