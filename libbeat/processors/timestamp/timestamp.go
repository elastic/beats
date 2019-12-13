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

package timestamp

import (
	"fmt"
	"time"

	"4d63.com/tz"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	jsprocessor "github.com/elastic/beats/libbeat/processors/script/javascript/module/processor"
)

const logName = "processor.timestamp"

func init() {
	processors.RegisterPlugin("timestamp", New)
	jsprocessor.RegisterPlugin("Timestamp", New)
}

type processor struct {
	config
	log     *logp.Logger
	isDebug bool
	tz      *time.Location
}

// New constructs a new timestamp processor for parsing time strings into
// time.Time values.
func New(cfg *common.Config) (processors.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, errors.Wrap(err, "failed to unpack the timestamp configuration")
	}

	return newFromConfig(c)
}

func newFromConfig(c config) (*processor, error) {
	loc, err := loadLocation(c.Timezone)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load timezone")
	}

	p := &processor{
		config:  c,
		log:     logp.NewLogger(logName),
		isDebug: logp.IsDebug(logName),
		tz:      loc,
	}
	if c.ID != "" {
		p.log = p.log.With("instance_id", c.ID)
	}

	// Execute user provided built-in tests.
	for _, test := range c.TestTimestamps {
		ts, err := p.parseValue(test)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse test timestamp")
		}
		p.log.Debugf("Test timestamp [%v] parsed as [%v].", test, ts.UTC())
	}

	return p, nil
}

var timezoneFormats = []string{"-07", "-0700", "-07:00"}

func loadLocation(timezone string) (*time.Location, error) {
	for _, format := range timezoneFormats {
		t, err := time.Parse(format, timezone)
		if err == nil {
			name, offset := t.Zone()
			return time.FixedZone(name, offset), nil
		}
	}

	// Rest of location formats
	return tz.LoadLocation(timezone)
}

func (p *processor) String() string {
	return fmt.Sprintf("timestamp=[field=%s, target_field=%v, timezone=%v]",
		p.Field, p.TargetField, p.tz)
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	// Get the source field value.
	val, err := event.GetValue(p.Field)
	if err != nil {
		if p.IgnoreFailure || (p.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound) {
			return event, nil
		}
		return event, errors.Wrapf(err, "failed to get time field %v", p.Field)
	}

	// Try to convert the value to a time.Time.
	ts, err := p.tryToTime(val)
	if err != nil {
		if p.IgnoreFailure {
			return event, nil
		}
		return event, err
	}

	// Put the timestamp as UTC into the target field.
	_, err = event.PutValue(p.TargetField, ts.UTC())
	if err != nil {
		if p.IgnoreFailure {
			return event, nil
		}
		return event, err
	}

	return event, nil
}

func (p *processor) tryToTime(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case common.Time:
		return time.Time(v), nil
	default:
		return p.parseValue(v)
	}
}

func (p *processor) parseValue(v interface{}) (time.Time, error) {
	detailedErr := &parseError{}

	for _, layout := range p.Layouts {
		ts, err := p.parseValueByLayout(v, layout)
		if err == nil {
			return ts, nil
		}

		switch e := err.(type) {
		case *time.ParseError:
			detailedErr.causes = append(detailedErr.causes, &parseErrorCause{e})
		default:
			detailedErr.causes = append(detailedErr.causes, err)
		}
	}

	detailedErr.field = p.Field
	detailedErr.time = v

	if p.isDebug {
		if p.IgnoreFailure {
			p.log.Debugw("(Ignored) Failure parsing time field.", "error", detailedErr)
		} else {
			p.log.Debugw("Failure parsing time field.", "error", detailedErr)
		}
	}
	return time.Time{}, detailedErr
}

func (p *processor) parseValueByLayout(v interface{}, layout string) (time.Time, error) {
	switch layout {
	case "UNIX":
		if sec, ok := common.TryToInt(v); ok {
			return time.Unix(int64(sec), 0), nil
		} else if sec, ok := common.TryToFloat64(v); ok {
			return time.Unix(0, int64(sec*float64(time.Second))), nil
		}
		return time.Time{}, errors.New("could not parse time field as int or float")
	case "UNIX_MS":
		if ms, ok := common.TryToInt(v); ok {
			return time.Unix(0, int64(ms)*int64(time.Millisecond)), nil
		} else if ms, ok := common.TryToFloat64(v); ok {
			return time.Unix(0, int64(ms*float64(time.Millisecond))), nil
		}
		return time.Time{}, errors.New("could not parse time field as int or float")
	default:
		str, ok := v.(string)
		if !ok {
			return time.Time{}, errors.Errorf("unexpected type %T for time field", v)
		}

		ts, err := time.ParseInLocation(layout, str, p.tz)
		if err == nil {
			// Use current year if no year is zero.
			if ts.Year() == 0 {
				currentYear := time.Now().In(ts.Location()).Year()
				ts = ts.AddDate(currentYear, 0, 0)
			}
		}
		return ts, err
	}
}
