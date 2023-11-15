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

package summarizertesthelper

// summarizertest exists to provide a helper function
// for the summarizer. We need a separate package to
// prevent import cycles.

import (
	"fmt"

	"github.com/elastic/beats/v7/heartbeat/hbtestllext"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer/jobsummary"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
	"github.com/elastic/go-lookslike/validator"
)

// This duplicates hbtest.SummaryChecks to avoid an import cycle.
// It could be refactored out, but it just isn't worth it.
func SummaryValidator(up uint16, down uint16) validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"summary":             summaryIsdef(up, down),
		"monitor.duration.us": hbtestllext.IsInt64,
	})
}

func summaryIsdef(up uint16, down uint16) isdef.IsDef {
	return isdef.Is("summary", func(path llpath.Path, v interface{}) *llresult.Results {
		js, ok := v.(jobsummary.JobSummary)
		if !ok {
			return llresult.SimpleResult(path, false, fmt.Sprintf("expected a *jobsummary.JobSummary, got %v", v))
		}

		if js.Up != up || js.Down != down {
			return llresult.SimpleResult(path, false, fmt.Sprintf("expected up/down to be %d/%d, got %d/%d", up, down, js.Up, js.Down))
		}

		return llresult.ValidResult(path)
	})
}
