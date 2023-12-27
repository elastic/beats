// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build windows

package wineventlog

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/andrewkroh/sys/windows/svc/eventlog"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestRenderer(t *testing.T) {
	logp.TestingSetup()

	t.Run(filepath.Base(sysmon9File), func(t *testing.T) {
		log := openLog(t, sysmon9File)
		defer log.Close()

		r, err := NewRenderer(NilHandle, logp.L())
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		events := renderAllEvents(t, log, r, true)
		assert.NotEmpty(t, events)

		if t.Failed() {
			logAsJSON(t, events)
		}
	})

	t.Run(filepath.Base(security4752File), func(t *testing.T) {
		log := openLog(t, security4752File)
		defer log.Close()

		r, err := NewRenderer(NilHandle, logp.L())
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		events := renderAllEvents(t, log, r, false)
		if !assert.Len(t, events, 1) {
			return
		}
		e := events[0]

		assert.EqualValues(t, 4752, e.EventIdentifier.ID)
		assert.Equal(t, "Microsoft-Windows-Security-Auditing", e.Provider.Name)
		assertEqualIgnoreCase(t, "{54849625-5478-4994-a5ba-3e3b0328c30d}", e.Provider.GUID)
		assert.Equal(t, "DC_TEST2k12.TEST.SAAS", e.Computer)
		assert.Equal(t, "Security", e.Channel)
		assert.EqualValues(t, 3707686, e.RecordID)

		assert.Equal(t, e.Keywords, []string{"Audit Success"})

		assert.NotNil(t, 0, e.OpcodeRaw)
		assert.EqualValues(t, 0, *e.OpcodeRaw)
		assert.Equal(t, "Info", e.Opcode)

		assert.EqualValues(t, 0, e.LevelRaw)
		assert.Equal(t, "Information", e.Level)

		assert.EqualValues(t, 13827, e.TaskRaw)
		assert.Equal(t, "Distribution Group Management", e.Task)

		assert.EqualValues(t, 492, e.Execution.ProcessID)
		assert.EqualValues(t, 1076, e.Execution.ThreadID)
		assert.Len(t, e.EventData.Pairs, 10)

		assert.NotEmpty(t, e.Message)

		if t.Failed() {
			logAsJSON(t, events)
		}
	})

	t.Run(filepath.Base(winErrorReportingFile), func(t *testing.T) {
		log := openLog(t, winErrorReportingFile)
		defer log.Close()

		r, err := NewRenderer(NilHandle, logp.L())
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		events := renderAllEvents(t, log, r, false)
		if !assert.Len(t, events, 1) {
			return
		}
		e := events[0]

		assert.EqualValues(t, 1001, e.EventIdentifier.ID)
		assert.Equal(t, "Windows Error Reporting", e.Provider.Name)
		assert.Empty(t, e.Provider.GUID)
		assert.Equal(t, "vagrant", e.Computer)
		assert.Equal(t, "Application", e.Channel)
		assert.EqualValues(t, 420107, e.RecordID)

		assert.Equal(t, e.Keywords, []string{"Classic"})

		assert.EqualValues(t, (*uint8)(nil), e.OpcodeRaw)
		assert.Equal(t, "", e.Opcode)

		assert.EqualValues(t, 4, e.LevelRaw)
		assert.Equal(t, "Information", e.Level)

		assert.EqualValues(t, 0, e.TaskRaw)
		assert.Equal(t, "None", e.Task)

		assert.EqualValues(t, 0, e.Execution.ProcessID)
		assert.EqualValues(t, 0, e.Execution.ThreadID)
		assert.Len(t, e.EventData.Pairs, 23)

		assert.NotEmpty(t, e.Message)

		if t.Failed() {
			logAsJSON(t, events)
		}
	})
}

func TestTemplateFunc(t *testing.T) {
	tmpl := template.Must(template.New("").
		Funcs(eventMessageTemplateFuncs).
		Parse(`Hello {{ eventParam $ 1 }}! Foo {{ eventParam $ 2 }}.`))

	buf := new(bytes.Buffer)
	err := tmpl.Execute(buf, []interface{}{"world"})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Hello world! Foo %2.", buf.String())
}

// renderAllEvents reads all events and renders them.
func renderAllEvents(t *testing.T, log EvtHandle, renderer *Renderer, ignoreMissingMetadataError bool) []*winevent.Event {
	t.Helper()

	var events []*winevent.Event
	for {
		h, done := nextHandle(t, log)
		if done {
			break
		}

		func() {
			defer h.Close()

			evt, err := renderer.Render(h)
			if err != nil {
				md := renderer.metadataCache[evt.Provider.Name]
				if !ignoreMissingMetadataError || md.Metadata != nil {
					t.Fatalf("Render failed: %+v", err)
				}
			}

			events = append(events, evt)
		}()
	}

	return events
}

// setLogSize set the maximum number of bytes that an event log can hold.
func setLogSize(t testing.TB, provider string, sizeBytes int) {
	output, err := exec.Command("wevtutil.exe", "sl", "/ms:"+strconv.Itoa(sizeBytes), provider).CombinedOutput() //nolint:gosec // No possibility of command injection.
	if err != nil {
		t.Fatal("failed to set log size", err, string(output))
	}
}

func BenchmarkRenderer(b *testing.B) {
	writer, teardown := createLog(b)
	defer teardown()

	const totalEvents = 1000000
	msg := []string{strings.Repeat("Hello world! ", 21)}
	for i := 0; i < totalEvents; i++ {
		safeWriteEvent(b, writer, eventlog.Info, 10, msg)
	}

	setup := func() (*EventIterator, *Renderer) {
		log := openLog(b, winlogbeatTestLogName)

		itr, err := NewEventIterator(WithSubscription(log), WithBatchSize(1024))
		if err != nil {
			log.Close()
			b.Fatal(err)
		}

		r, err := NewRenderer(NilHandle, logp.NewLogger("bench"))
		if err != nil {
			log.Close()
			itr.Close()
			b.Fatal(err)
		}

		return itr, r
	}

	b.Run("single_thread", func(b *testing.B) {
		itr, r := setup()
		defer itr.Close()
		defer r.Close()

		count := atomic.NewUint64(0)
		start := time.Now()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Get next handle.
			h, ok := itr.Next()
			if !ok {
				b.Fatal("Ran out of events before benchmark was done.", itr.Err())
			}

			// Render it.
			_, err := r.Render(h)
			if err != nil {
				b.Fatal(err)
			}

			count.Inc()
		}

		elapsed := time.Since(start)
		b.ReportMetric(float64(count.Load())/elapsed.Seconds(), "events/sec")
	})

	b.Run("parallel8", func(b *testing.B) {
		itr, r := setup()
		defer itr.Close()
		defer r.Close()

		count := atomic.NewUint64(0)
		start := time.Now()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Get next handle.
				h, ok := itr.Next()
				if !ok {
					b.Fatal("Ran out of events before benchmark was done.", itr.Err())
				}

				// Render it.
				_, err := r.Render(h)
				if err != nil {
					b.Fatal(err)
				}
				count.Inc()
			}
		})

		elapsed := time.Since(start)
		b.ReportMetric(float64(count.Load())/elapsed.Seconds(), "events/sec")
		b.ReportMetric(float64(runtime.GOMAXPROCS(0)), "gomaxprocs")
	})
}
