package metrics

type cpu struct {
	User uint64
	Nice uint64
	Sys  uint64
	Idle uint64
}

type cpuMetrics struct {
	totals cpu
	list   []cpu
}

func (self cpuMetrics) Total() uint64 {
	return self.totals.User + self.totals.Nice + self.totals.Sys + self.totals.Idle
}

func (self cpuMetrics) FillTicks(event *common.MapStr) {
	event.Put("user.ticks", self.totals.User)
	event.Put("system.ticks", self.totals.Sys)
	event.Put("idle.ticks", self.totals.Idle)
	event.Put("nice.ticks", self.totals.Nice)
}

func fillCPUMetrics(event *common.MapStr, current, prev cpuMetrics, numCPU int, timeDelta uint64, pathPostfix string) {
	// IOWait time is excluded from the total as per #7627.
	idleTime := cpuMetricTimeDelta(prev.totals.Idle, current.totals.Idle, timeDelta, numCPU) + cpuMetricTimeDelta(prev.totals.Wait, current.totals.Wait, timeDelta, numCPU)
	totalPct := common.Round(float64(numCPU)-idleTime, common.DefaultDecimalPlacesCount)

	event.Put("total"+pathPostfix, totalPct)
	event.Put("user"+pathPostfix, cpuMetricTimeDelta(prev.totals.User, current.totals.User, timeDelta, numCPU))
	event.Put("system"+pathPostfix, cpuMetricTimeDelta(prev.totals.Sys, current.totals.Sys, timeDelta, numCPU))
	event.Put("idle"+pathPostfix, cpuMetricTimeDelta(prev.totals.Idle, current.totals.Idle, timeDelta, numCPU))
	event.Put("nice"+pathPostfix, cpuMetricTimeDelta(prev.totals.Nice, current.totals.Nice, timeDelta, numCPU))
}
