package monitoring

import (
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/metric/system/cpu"
	"github.com/elastic/beats/libbeat/metric/system/process"
)

const samplingInterval = 30 * time.Second

// Collector collects metrics from the Beat and its host periodically,
// so stateful metrics can be calculated and retrieved properly
type Collector struct {
	sync.RWMutex
	wg   sync.WaitGroup
	done chan struct{}

	cpu          *cpu.Monitor
	processStats *process.Stats

	cpuUsage      common.MapStr
	processInfo   common.MapStr
	systemPct     cpu.Percentages
	systemNormPct cpu.Percentages
}

// MakeCollector creates a collector instance and starts collecting data periodically
func MakeCollector(name string) (*Collector, error) {
	c := &Collector{
		cpu:  new(cpu.Monitor),
		done: make(chan struct{}, 1),
		processStats: &process.Stats{
			Procs:        []string{name},
			EnvWhitelist: nil,
			CpuTicks:     false,
			CacheCmdLine: true,
			IncludeTop:   process.IncludeTopConfig{},
		},
	}

	err := c.processStats.Init()
	if err != nil {
		return nil, err
	}
	c.collectSystemCPU()
	c.collectProcessStats()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.update()
	}()

	return c, err
}

func (c *Collector) update() {
	logp.Info("Start collecting system and Beat mertics")
	defer logp.Info("Stopping metrics collector")

	ticker := time.NewTicker(samplingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.collectSystemCPU()
			c.collectProcessStats()
		}
	}
}

func (c *Collector) collectSystemCPU() {
	c.Lock()
	defer c.Unlock()

	sample, err := c.cpu.Sample()
	if err != nil {
		logp.Err("Error retrieving CPU usage of the host: %v", err)
		return
	}

	c.systemPct = sample.Percentages()
	c.systemNormPct = sample.NormalizedPercentages()
}

func (c *Collector) collectProcessStats() {
	c.Lock()
	defer c.Unlock()

	beatPID := os.Getpid()
	state, err := c.processStats.GetOne(beatPID)
	if err != nil {
		logp.Err("Error retrieving process stats of Beat")
		return
	}

	c.processInfo = state
	c.cpuUsage = state
}

// Stop stops metrics collector
func (c *Collector) Stop() {
	close(c.done)
	c.wg.Wait()
}

// CPUInfo retrieves the CPU usage of the Beat
func (c *Collector) CPUInfo() common.MapStr {
	c.RLock()
	defer c.RUnlock()

	return c.cpuUsage
}

// ProcessInfo retrieves the process info of the Beat
func (c *Collector) ProcessInfo() common.MapStr {
	c.RLock()
	defer c.RUnlock()

	return c.processInfo
}

// SystemCPUInfo retrieves the CPU usage of the host
func (c *Collector) SystemCPUInfo() (cpu.Percentages, cpu.Percentages) {
	c.RLock()
	defer c.RUnlock()

	return c.systemPct, c.systemNormPct
}
