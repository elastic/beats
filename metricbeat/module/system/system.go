package system

import (
	"flag"
	"math"
	"sync"

	"github.com/elastic/beats/metricbeat/mb"
)

var (
	HostFS = flag.String("system.hostfs", "", "mountpoint of the host's filesystem for use in monitoring a host from within a container")
)

var once sync.Once

func init() {
	// Register the ModuleFactory function for the "system" module.
	if err := mb.Registry.AddModule("system", NewModule); err != nil {
		panic(err)
	}
}

type Module struct {
	mb.BaseModule
	HostFS string // Mountpoint of the host's filesystem for use in monitoring inside a container.
}

func NewModule(base mb.BaseModule) (mb.Module, error) {
	// This only needs to be configured once for all system modules.
	once.Do(func() {
		configureHostFS()
	})

	return &Module{BaseModule: base, HostFS: *HostFS}, nil
}

func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}
