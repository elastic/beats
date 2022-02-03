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
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

type addLocale struct {
	TimezoneFormat TimezoneFormat
}

// TimezoneFormat type
type TimezoneFormat int

// Timezone formats
const (
	Abbreviation TimezoneFormat = iota
	Offset
)

var timezoneFormats = map[TimezoneFormat]string{
	Abbreviation: "abbreviation",
	Offset:       "offset",
}

func (t TimezoneFormat) String() string {
	return timezoneFormats[t]
}

func init() {
	processors.RegisterPlugin("add_locale", New)
	jsprocessor.RegisterPlugin("AddLocale", New)
}

// New constructs a new add_locale processor.
func New(c *common.Config) (processors.Processor, error) {
	config := struct {
		Format string `config:"format"`
	}{
		Format: "offset",
	}

	err := c.Unpack(&config)
	if err != nil {
		return nil, errors.Wrap(err, "fail to unpack the add_locale configuration")
	}

	var loc addLocale

	switch strings.ToLower(config.Format) {
	case "abbreviation":
		loc.TimezoneFormat = Abbreviation
	case "offset":
		loc.TimezoneFormat = Offset
	default:
		return nil, errors.Errorf("'%s' is not a valid format option for the "+
			"add_locale processor. Valid options are 'abbreviation' and 'offset'.",
			config.Format)

	}
	return loc, nil
}

func (l addLocale) Run(event *beat.Event) (*beat.Event, error) {
	zone, offset := time.Now().Zone()
	format := l.Format(zone, offset)
	event.Fields.Put("event.timezone", format)
	return event, nil
}

const (
	sec  = 1
	min  = 60 * sec
	hour = 60 * min
)

func (l addLocale) Format(zone string, offset int) string {
	var ft string
	switch l.TimezoneFormat {
	case Abbreviation:
		ft = zone
	case Offset:
		sign := "+"
		if offset < 0 {
			sign = "-"
			offset *= -1
		}

		h := offset / hour
		m := (offset - (h * hour)) / min
		ft = fmt.Sprintf("%s%02d:%02d", sign, h, m)
	}
	return ft
}

func (l addLocale) String() string {
	return "add_locale=[format=" + l.TimezoneFormat.String() + "]"
}
