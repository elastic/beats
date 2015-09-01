package main

import ()

type TopConfig struct {
	Period *int64
	Procs  *[]string
}

type ConfigSettings struct {
	Input TopConfig
}

var Config ConfigSettings
