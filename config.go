package main

type TopConfig struct {
	Period     *int64
	Procs      *[]string
	Stats struct {
		System     *bool
		Proc       *bool
		Filesystem *bool
	}
}

type ConfigSettings struct {
	Input TopConfig
}
