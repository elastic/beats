package logp

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/monitoring"
)

type snapshotVisitor struct {
	snapshot snapshot
	level    []string
}

type snapshot struct {
	bools   map[string]bool
	ints    map[string]int64
	floats  map[string]float64
	strings map[string]string
}

// logMetrics logs at Info level the integer expvars that have changed in the
// last interval. For each expvar, the delta from the beginning of the interval
// is logged.
func logMetrics(metricsCfg *LoggingMetricsConfig) {
	if metricsCfg.Enabled != nil && *metricsCfg.Enabled == false {
		Info("Metrics logging disabled")
		return
	}
	if metricsCfg.Period == nil {
		metricsCfg.Period = &defaultMetricsPeriod
	}
	Info("Metrics logging every %s", metricsCfg.Period)

	ticker := time.NewTicker(*metricsCfg.Period)

	prevVals := makeSnapshot()
	for range ticker.C {
		snapshot := snapshotMetrics()
		delta := snapshotDelta(prevVals, snapshot)
		prevVals = snapshot

		if len(delta) == 0 {
			Info("No non-zero metrics in the last %s", metricsCfg.Period)
			continue
		}

		metrics := formatMetrics(delta)
		Info("Non-zero metrics in the last %s:%s", metricsCfg.Period, metrics)
	}
}

// LogTotalExpvars logs all registered expvar metrics.
func LogTotalExpvars(cfg *Logging) {
	if cfg.Metrics.Enabled != nil && *cfg.Metrics.Enabled == false {
		return
	}

	metrics := formatMetrics(snapshotDelta(makeSnapshot(), snapshotMetrics()))
	Info("Total non-zero values: %s", metrics)
	Info("Uptime: %s", time.Now().Sub(startTime))
}

func snapshotMetrics() snapshot {
	vs := newSnapshotVisitor()
	monitoring.Default.Visit(vs)
	monitoring.VisitExpvars(vs)
	return vs.snapshot
}

func newSnapshotVisitor() *snapshotVisitor {
	return &snapshotVisitor{snapshot: makeSnapshot()}
}

func makeSnapshot() snapshot {
	return snapshot{
		bools:   map[string]bool{},
		ints:    map[string]int64{},
		floats:  map[string]float64{},
		strings: map[string]string{},
	}
}

func (vs *snapshotVisitor) OnRegistryStart() error {
	return nil
}

func (vs *snapshotVisitor) OnRegistryFinished() error {
	if len(vs.level) > 0 {
		vs.dropName()
	}
	return nil
}

func (vs *snapshotVisitor) OnKey(name string) error {
	vs.level = append(vs.level, name)
	return nil
}

func (vs *snapshotVisitor) OnKeyNext() error { return nil }

func (vs *snapshotVisitor) getName() string {
	defer vs.dropName()
	if len(vs.level) == 1 {
		return vs.level[0]
	}
	return strings.Join(vs.level, ".")
}

func (vs *snapshotVisitor) dropName() {
	vs.level = vs.level[:len(vs.level)-1]
}

func (vs *snapshotVisitor) OnString(s string) error {
	vs.snapshot.strings[vs.getName()] = s
	return nil
}

func (vs *snapshotVisitor) OnBool(b bool) error {
	vs.snapshot.bools[vs.getName()] = b
	return nil
}

func (vs *snapshotVisitor) OnNil() error {
	vs.snapshot.strings[vs.getName()] = "<nil>"
	return nil
}

func (vs *snapshotVisitor) OnInt(i int64) error {
	vs.snapshot.ints[vs.getName()] = i
	return nil
}

func (vs *snapshotVisitor) OnFloat(f float64) error {
	vs.snapshot.floats[vs.getName()] = f
	return nil
}

func snapshotDelta(prev, cur snapshot) map[string]interface{} {
	out := map[string]interface{}{}

	for k, b := range cur.bools {
		if p, ok := prev.bools[k]; !ok || p != b {
			out[k] = b
		}
	}

	for k, s := range cur.strings {
		if p, ok := prev.strings[k]; !ok || p != s {
			out[k] = s
		}
	}

	for k, i := range cur.ints {
		if p := prev.ints[k]; p != i {
			out[k] = i - p
		}
	}

	for k, f := range cur.floats {
		if p := prev.floats[k]; p != f {
			out[k] = f - p
		}
	}

	return out
}

func formatMetrics(ms map[string]interface{}) string {
	keys := make([]string, 0, len(ms))
	for key := range ms {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	var buf bytes.Buffer
	for _, key := range keys {
		buf.WriteByte(' ')
		buf.WriteString(key)
		buf.WriteString("=")
		buf.WriteString(fmt.Sprintf("%v", ms[key]))
	}
	return buf.String()
}
