package log

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/monitoring"
)

var (
	prevSnap = monitoring.FlatSnapshot{
		Ints: map[string]int64{
			"count": 10,
			"gone":  1,
		},
	}
	curSnap = monitoring.FlatSnapshot{
		Ints: map[string]int64{
			"count": 20,
			"new":   1,
		},
	}
)

type fakeLogger struct {
	logs []string
}

func (l *fakeLogger) Infof(format string, v ...interface{}) {
	l.logs = append(l.logs, fmt.Sprintf(format, v...))
}

// Smoke test.
func TestStartStop(t *testing.T) {
	r, err := MakeReporter(beat.Info{}, common.NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	r.Stop()
}

func TestMakeDeltaSnapshot(t *testing.T) {
	delta := makeDeltaSnapshot(prevSnap, curSnap)
	assert.EqualValues(t, 10, delta.Ints["count"])
	assert.EqualValues(t, 1, delta.Ints["new"])
	assert.NotContains(t, delta.Ints, "gone")
}

func TestReporterLog(t *testing.T) {
	logger := &fakeLogger{}
	reporter := reporter{period: 30 * time.Second, logger: logger}

	reporter.logSnapshot(monitoring.FlatSnapshot{})
	assert.Equal(t, "No non-zero metrics in the last 30s", logger.logs[0])

	reporter.logSnapshot(
		monitoring.FlatSnapshot{
			Bools: map[string]bool{
				"running": true,
			},
		},
	)
	assert.Equal(t, "Non-zero metrics in the last 30s: running=true", logger.logs[1])

	reporter.logTotals(curSnap)
	assert.Equal(t, "Total non-zero metrics: count=20 new=1", logger.logs[2])
	assert.Contains(t, logger.logs[3], "Uptime: ")
}
