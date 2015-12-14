package beat

type TopConfig struct {
	Period *int64
	Procs  *[]string
	Stats  struct {
		System     *bool `yaml:"system"`
		Proc       *bool `yaml:"process"`
		Filesystem *bool `yaml:"filesystem"`
		CpuPerCore *bool `yaml:"cpu_per_core"`
	}
}

type ConfigSettings struct {
	Input TopConfig
}
