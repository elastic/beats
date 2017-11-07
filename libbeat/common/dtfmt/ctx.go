package dtfmt

import (
	"time"
)

// ctx stores pre-computed time fields used by the formatter.
type ctx struct {
	year             int
	month            time.Month
	day              int
	weekday          time.Weekday
	yearday          int
	isoWeek, isoYear int

	hour, min, sec int
	millis         int

	buf []byte
}

type ctxConfig struct {
	date    bool
	clock   bool
	weekday bool
	yearday bool
	millis  bool
	iso     bool
}

func (c *ctx) initTime(config *ctxConfig, t time.Time) {
	if config.date {
		c.year, c.month, c.day = t.Date()
	}
	if config.clock {
		c.hour, c.min, c.sec = t.Clock()
	}
	if config.iso {
		c.isoYear, c.isoWeek = t.ISOWeek()
	}

	if config.millis {
		c.millis = t.Nanosecond() / 1000000
	}

	if config.yearday {
		c.yearday = t.YearDay()
	}

	if config.weekday {
		c.weekday = t.Weekday()
	}
}

func (c *ctxConfig) enableDate() {
	c.date = true
}

func (c *ctxConfig) enableClock() {
	c.clock = true
}

func (c *ctxConfig) enableMillis() {
	c.millis = true
}

func (c *ctxConfig) enableWeekday() {
	c.weekday = true
}

func (c *ctxConfig) enableYearday() {
	c.yearday = true
}

func (c *ctxConfig) enableISO() {
	c.iso = true
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
