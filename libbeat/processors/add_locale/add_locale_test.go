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

package add_locale

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func TestExportTimezone(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"format": "abbreviation",
	})
	if err != nil {
		t.Fatal(err)
	}

	input := common.MapStr{}

	zone, _ := time.Now().In(time.Local).Zone()

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"event": map[string]string{
			"timezone": zone,
		},
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestTimezoneFormat(t *testing.T) {
	// Test positive format

	posLoc, err := time.LoadLocation("Africa/Asmara")
	if err != nil {
		t.Fatal(err)
	}

	posZone, posOffset := time.Now().In(posLoc).Zone()

	posAddLocal := addLocale{TimezoneFormat: Offset}

	posVal := posAddLocal.Format(posZone, posOffset)

	assert.Regexp(t, regexp.MustCompile(`\+[\d]{2}\:[\d]{2}`), posVal)

	// Test negative format

	negLoc, err := time.LoadLocation("America/Curacao")
	if err != nil {
		t.Fatal(err)
	}

	negZone, negOffset := time.Now().In(negLoc).Zone()

	negAddLocal := addLocale{TimezoneFormat: Offset}

	negVal := negAddLocal.Format(negZone, negOffset)

	assert.Regexp(t, regexp.MustCompile(`\-[\d]{2}\:[\d]{2}`), negVal)
}

func getActualValue(t *testing.T, config *common.Config, input common.MapStr) common.MapStr {
	logp.TestingSetup()

	p, err := New(config)
	if err != nil {
		logp.Err("Error initializing add_locale")
		t.Fatal(err)
	}

	actual, err := p.Run(&beat.Event{Fields: input})
	return actual.Fields
}

func BenchmarkConstruct(b *testing.B) {
	var testConfig = common.NewConfig()

	input := common.MapStr{}

	p, err := New(testConfig)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, err = p.Run(&beat.Event{Fields: input})
	}
}
