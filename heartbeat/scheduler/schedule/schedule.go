package schedule

import (
	"errors"
	"strings"
	"time"

	"github.com/elastic/beats/heartbeat/scheduler"
	"github.com/elastic/beats/heartbeat/scheduler/schedule/cron"
)

type Schedule struct {
	scheduler.Schedule
}

type intervalScheduler struct {
	interval time.Duration
}

func Parse(in string) (*Schedule, error) {
	every := "@every"

	// add '@every' keyword
	if strings.HasPrefix(in, every) {
		interval := strings.TrimSpace(in[len(every):])
		d, err := time.ParseDuration(interval)
		if err != nil {
			return nil, err
		}

		return &Schedule{intervalScheduler{d}}, nil
	}

	// fallback on cron scheduler parsers
	s, err := cron.Parse(in)
	if err != nil {
		return nil, err
	}
	return &Schedule{s}, nil
}

func (s intervalScheduler) Next(t time.Time) time.Time {
	return t.Add(s.interval)
}

func (s *Schedule) Unpack(in interface{}) error {
	str, ok := in.(string)
	if !ok {
		return errors.New("scheduler string required")
	}

	tmp, err := Parse(str)
	if err != nil {
		return err
	}

	*s = *tmp
	return nil
}
