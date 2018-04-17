// +build windows

package eventlog

import (
	"bytes"
	"flag"
	"math/rand"
	"os/exec"
	"strconv"
	"testing"
	"time"

	elog "github.com/andrewkroh/sys/windows/svc/eventlog"
	"github.com/dustin/go-humanize"
)

// Benchmark tests with customized output. (`go test -v -benchtime 10s -benchtest .`)

var (
	benchTest    = flag.Bool("benchtest", false, "Run benchmarks for the eventlog package")
	injectAmount = flag.Int("inject", 50000, "Number of events to inject before running benchmarks")
)

// TestBenchmarkBatchReadSize tests the performance of different
// batch_read_size values.
func TestBenchmarkBatchReadSize(t *testing.T) {
	if !*benchTest {
		t.Skip("-benchtest not enabled")
	}

	log, err := initLog(providerName, sourceName, eventCreateMsgFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := uninstallLog(providerName, sourceName, log)
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Increase the log size so that it can hold these large events.
	output, err := exec.Command("wevtutil.exe", "sl", "/ms:1073741824", providerName).CombinedOutput()
	if err != nil {
		t.Fatal(err, string(output))
	}

	// Publish test messages:
	for i := 0; i < *injectAmount; i++ {
		err = log.Report(elog.Info, uint32(rand.Int63()%1000), []string{strconv.Itoa(i) + " " + randomSentence(256)})
		if err != nil {
			t.Fatal("ReportEvent error", err)
		}
	}

	benchTest := func(batchSize int) {
		var err error
		result := testing.Benchmark(func(b *testing.B) {
			eventlog, tearDown := setupWinEventLog(t, 0, map[string]interface{}{
				"name":            providerName,
				"batch_read_size": batchSize,
			})
			defer tearDown()
			b.ResetTimer()

			// Each iteration reads one batch.
			for i := 0; i < b.N; i++ {
				_, err = eventlog.Read()
				if err != nil {
					return
				}
			}
		})

		if err != nil {
			t.Fatal(err)
			return
		}

		t.Logf("batch_size=%v, total_events=%v, batch_time=%v, events_per_sec=%v, bytes_alloced_per_event=%v, total_allocs=%v",
			batchSize,
			result.N*batchSize,
			time.Duration(result.NsPerOp()),
			float64(batchSize)/time.Duration(result.NsPerOp()).Seconds(),
			humanize.Bytes(result.MemBytes/(uint64(result.N)*uint64(batchSize))),
			result.MemAllocs)
	}

	benchTest(10)
	benchTest(100)
	benchTest(500)
	benchTest(1000)
}

// Utility Functions

var randomWords = []string{
	"recover",
	"article",
	"highway",
	"bargain",
	"trolley",
	"college",
	"attract",
	"wriggle",
	"feather",
	"neutral",
	"percent",
	"quality",
	"manager",
	"hunting",
	"arrange",
}

func randomSentence(n uint) string {
	buf := bytes.NewBuffer(make([]byte, n))
	buf.Reset()

	for {
		idx := rand.Uint32() % uint32(len(randomWords))
		word := randomWords[idx]

		if buf.Len()+len(word) <= buf.Cap() {
			buf.WriteString(randomWords[idx])
		} else {
			break
		}

		if buf.Len()+1 <= buf.Cap() {
			buf.WriteByte(' ')
		} else {
			break
		}
	}

	return buf.String()
}
