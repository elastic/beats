package monitoring

import (
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
)

var (
	// Process start time.
	startTime time.Time
)

func init() {
	startTime = time.Now()
}

type Config struct {
	Enabled *bool         `config:"logging.metrics.enabled"`
	Period  time.Duration `config:"logging.metrics.period" validate:"nonzero,min=0s"`
}

func (c *Config) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}

var defaultConfig = Config{
	Period: 30 * time.Second,
}

type PeriodicLogger struct {
	period time.Duration
	logger *logp.Logger

	stopOnce sync.Once
	done     chan struct{}
	wg       sync.WaitGroup
}

func StartNewPeriodLogger(period time.Duration) (*PeriodicLogger, error) {
	if period <= 0 {
		return nil, errors.New("metrics logging period must be greater than 0")
	}

	l := &PeriodicLogger{
		period: period,
		logger: logp.NewLogger("metrics"),
		done:   make(chan struct{}),
	}
	go l.run()
	return l, nil
}

func (l *PeriodicLogger) run() {
	ticker := time.NewTicker(l.period)
	defer ticker.Stop()

	logp.Info("Metrics logging every %s", l.period)

	prevVals := MakeFlatSnapshot()
	for {
		select {
		case <-ticker.C:
			snapshot := snapshotMetrics()
			delta := snapshotDelta(prevVals, snapshot)
			prevVals = snapshot

			if len(delta) == 0 {
				logp.Info("No non-zero metrics in the last %s", l.period)
				continue
			}

			metrics := append([]logp.Field{
				logp.Duration("interval", l.period),
				logp.Namespace("metrics"),
			}, delta...)
			l.logger.Info("Non-zero metrics in last interval", metrics...)
		case <-l.done:
			return
		}
	}
}

func (l *PeriodicLogger) Stop() {
	l.stopOnce.Do(func() {
		close(l.done)
		l.wg.Wait()
		l.logTotals()
	})
}

func (l *PeriodicLogger) logTotals() {
	zero := MakeFlatSnapshot()
	metrics := snapshotDelta(zero, snapshotMetrics())
	l.logger.Info("Total non-zero metrics", metrics...)
	l.logger.Info("Total uptime", logp.Duration("uptime.ms", time.Since(startTime)))
}

func snapshotMetrics() FlatSnapshot {
	return CollectFlatSnapshot(Default, Full, true)
}

// List of metrics that are gauges, so that we know for which to
// _not_ subtract the previous value in the output.
// TODO: Replace this with a proper solution that uses the metric
// type from where it is defined. See:
// https://github.com/elastic/beats/issues/5433
var gauges = map[string]bool{
	"libbeat.pipeline.events.active": true,
	"libbeat.pipeline.clients":       true,
	"libbeat.config.module.running":  true,
	"registrar.states.current":       true,
	"filebeat.harvester.running":     true,
	"filebeat.harvester.open_files":  true,
	"beat.memstats.memory_total":     true,
	"beat.memstats.memory_alloc":     true,
	"beat.memstats.gc_next":          true,
}

func snapshotDelta(prev, cur FlatSnapshot) []logp.Field {
	var fields []logp.Field

	fields = append(fields, logp.Namespace("metrics"))

	for k, b := range cur.Bools {
		if p, ok := prev.Bools[k]; !ok || p != b {
			fields = append(fields, logp.Bool(k, b))
		}
	}

	for k, s := range cur.Strings {
		if p, ok := prev.Strings[k]; !ok || p != s {
			fields = append(fields, logp.String(k, s))
		}
	}

	for k, i := range cur.Ints {
		if _, found := gauges[k]; found {
			fields = append(fields, logp.Int64(k, i))
		} else {
			if p := prev.Ints[k]; p != i {
				fields = append(fields, logp.Int64(k, i-p))
			}
		}
	}

	for k, f := range cur.Floats {
		if p := prev.Floats[k]; p != f {
			fields = append(fields, logp.Float64(k, f-p))
		}
	}

	return fields
}
