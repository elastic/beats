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

package secretstores

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

// redactorFor returns a Redactor of the given secret values.
func redactorFor(values ...string) *Redactor {
	r := NewRedactor()
	r.set(0, scrubberFor(values...))
	return r
}

func TestRedactorRun(t *testing.T) {
	event := &beat.Event{Fields: mapstr.M{"url": mapstr.M{"full": "https://host/?t=s3cr3t-token"}}}
	out, err := redactorFor("s3cr3t-token").Run(event)
	assert.NoError(t, err, "the redactor never fails")
	assert.Equal(t, mapstr.M{"url": mapstr.M{"full": "https://host/?t=<hidden>"}}, out.Fields, "the event must not have the secret")

	event = &beat.Event{Fields: mapstr.M{"message": "s3cr3t-token"}}
	_, _ = NewRedactor().Run(event)
	assert.Equal(t, mapstr.M{"message": "s3cr3t-token"}, event.Fields, "a redactor without values must leave the event as is")
}

// TestRedactorCoversEveryJob checks the values of every job's latest run are
// removed: the jobs of a monitor with several endpoints pick up a rotated
// secret one after the other, and can publish at the same time.
func TestRedactorCoversEveryJob(t *testing.T) {
	r := NewRedactor()
	r.set(0, scrubberFor("first-secret"))
	r.set(1, scrubberFor("first-secret"))

	r.set(0, scrubberFor("second-secret"))
	assert.Equal(t, "<hidden> <hidden>", r.scrub("first-secret second-secret"), "job 1 still runs with the first secret")

	// A run that fails before any value is substituted publishes none.
	r.set(0, nil)
	assert.Equal(t, "<hidden> second-secret", r.scrub("first-secret second-secret"), "only the values of the latest runs must be removed")

	r.set(1, scrubberFor("second-secret"))
	assert.Equal(t, "first-secret <hidden>", r.scrub("first-secret second-secret"), "the first secret is no longer in any run")
}

// TestRedactorBrowserError checks the error a browser monitor publishes. The
// summarizer puts the ECS error of the run in the event, and the pipeline
// converts it to a map before the redactor runs.
func TestRedactorBrowserError(t *testing.T) {
	const secret = "s3cr3t-token"
	stack := "at login (token=" + secret + ")"
	err := ecserr.NewECSErrWithStack("journey", "FAILED", "failed with "+secret, &stack)

	event := &beat.Event{Fields: mapstr.M{}}
	summarizer.NewBrowserErrPlugin().EachEvent(event, err)
	event.Fields = common.NewGenericEventConverter(false, false, logptest.NewTestingLogger(t, "")).Convert(event.Fields)
	_, _ = redactorFor(secret).Run(event)

	msg, _ := event.GetValue("error.message")
	assert.Equal(t, "failed with <hidden>", msg, "error.message must not have the secret")
	trace, _ := event.GetValue("error.stack_trace")
	assert.Equal(t, "at login (token=<hidden>)", trace, "error.stack_trace must not have the secret")
}
