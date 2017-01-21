// +build windows

package eventlog

import (
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
		err = log.Report(elog.Info, uint32(rng.Int63()%1000), []string{strconv.Itoa(i) + " " + randString(256)})
		if err != nil {
			t.Fatal("ReportEvent error", err)
		}
	}

	setup := func(t testing.TB, batchReadSize int) (EventLog, func()) {
		eventlog, err := newWinEventLog(map[string]interface{}{"name": providerName, "batch_read_size": batchReadSize})
		if err != nil {
			t.Fatal(err)
		}
		err = eventlog.Open(0)
		if err != nil {
			t.Fatal(err)
		}
		return eventlog, func() {
			err := eventlog.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	benchTest := func(batchSize int) {
		var err error
		result := testing.Benchmark(func(b *testing.B) {
			eventlog, tearDown := setup(b, batchSize)
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

var rng = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func randString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, rng.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rng.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
