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

package file

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type intervalRotator struct {
	log        Logger
	interval   time.Duration
	lastRotate time.Time
	filename   string
	fileFormat string
	clock      clock
	weekly     bool
	arbitrary  bool
}

func newIntervalRotator(log Logger, interval time.Duration, filename string) rotater {
	ir := &intervalRotator{
		filename: filename,
		log:      log,
		interval: (interval / time.Second) * time.Second, // drop fractional seconds
		clock:    realClock{},
	}
	ir.initialize()
	return ir
}

func (r *intervalRotator) initialize() {
	switch r.interval {
	case time.Second:
		r.fileFormat = "2006-01-02-15-04-05"
	case time.Minute:
		r.fileFormat = "2006-01-02-15-04"
	case time.Hour:
		r.fileFormat = "2006-01-02-15"
	case 24 * time.Hour: // calendar day
		r.fileFormat = "2006-01-02"
	case 7 * 24 * time.Hour: // calendar week
		r.fileFormat = ""
		r.weekly = true
	case 30 * 24 * time.Hour: // calendar month
		r.fileFormat = "2006-01"
	case 365 * 24 * time.Hour: // calendar year
		r.fileFormat = "2006"
	default:
		r.arbitrary = true
		r.fileFormat = "2006-01-02-15-04-05"
	}

	fi, err := os.Stat(r.filename)
	if err != nil {
		if r.log != nil {
			r.log.Debugw("Not attempting to find last rotated time, configured logs dir cannot be opened: %v", err)
		}
		return
	}
	r.lastRotate = fi.ModTime()
}

func (r *intervalRotator) ActiveFile() string {
	return r.filename
}

func (r *intervalRotator) LogPrefix(filename string, modTime time.Time) string {
	var t time.Time
	if r.lastRotate.IsZero() {
		t = modTime
	} else {
		t = r.lastRotate
	}

	if r.weekly {
		y, w := t.ISOWeek()
		return fmt.Sprintf("%s-%04d-%02d-", filename, y, w)
	}
	if r.arbitrary {
		intervalNumber := t.Unix() / (int64(r.interval) / int64(time.Second))
		intervalStart := time.Unix(0, intervalNumber*int64(r.interval))
		return fmt.Sprintf("%s-%s-", filename, intervalStart.Format(r.fileFormat))
	}
	return fmt.Sprintf("%s-%s-", filename, t.Format(r.fileFormat))
}

func (r *intervalRotator) RotatedFiles() []string {
	files, err := filepath.Glob(r.filename + "*")
	if err != nil {
		if r.log != nil {
			r.log.Debugw("failed to list existing logs: %+v", err)
		}
	}
	r.SortIntervalLogs(files)
	return files
}

func (r *intervalRotator) Rotate(reason rotateReason, t time.Time) error {
	fi, err := os.Stat(r.ActiveFile())
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to rotate backups")
	}

	logPrefix := r.LogPrefix(r.ActiveFile(), fi.ModTime())
	files, err := filepath.Glob(logPrefix + "*")
	if err != nil {
		return errors.Wrap(err, "failed to list logs during rotation")
	}

	var targetFilename string
	if len(files) == 0 {
		targetFilename = logPrefix + "1"
	} else {
		r.SortIntervalLogs(files)
		lastLogIndex, _, err := IntervalLogIndex(files[len(files)-1])
		if err != nil {
			return errors.Wrap(err, "failed to locate last log index during rotation")
		}
		targetFilename = logPrefix + strconv.Itoa(int(lastLogIndex)+1)
	}

	if err := os.Rename(r.ActiveFile(), targetFilename); err != nil {
		return errors.Wrap(err, "failed to rotate backups")
	}

	if r.log != nil {
		r.log.Debugw("Rotating file", "filename", r.ActiveFile(), "reason", reason)
	}

	r.lastRotate = t
	return nil
}

func (r *intervalRotator) SortIntervalLogs(strings []string) {
	sort.Slice(
		strings,
		func(i, j int) bool {
			return OrderIntervalLogs(strings[i]) < OrderIntervalLogs(strings[j])
		},
	)
}

// OrderIntervalLogs, when given a log filename in the form [prefix]-[formattedDate]-n
// returns the filename after zero-padding the trailing n so that foo-[date]-2 sorts
// before foo-[date]-10.
func OrderIntervalLogs(filename string) string {
	index, i, err := IntervalLogIndex(filename)
	if err == nil {
		return filename[:i] + fmt.Sprintf("%020d", index)
	}

	return ""
}

// IntervalLogIndex returns n as int given a log filename in the form [prefix]-[formattedDate]-n
func IntervalLogIndex(filename string) (uint64, int, error) {
	i := len(filename) - 1
	for ; i >= 0; i-- {
		if '0' > filename[i] || filename[i] > '9' {
			break
		}
	}
	i++

	s64 := filename[i:]
	u64, err := strconv.ParseUint(s64, 10, 64)
	return u64, i, err
}
