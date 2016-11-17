package cron

import (
	"errors"
	"time"

	"github.com/gorhill/cronexpr"
)

type Schedule cronexpr.Expression

func MustParse(in string) *Schedule {
	s, err := Parse(in)
	if err != nil {
		panic(err)
	}
	return s
}

func Parse(in string) (*Schedule, error) {
	expr, err := cronexpr.Parse(in)
	return (*Schedule)(expr), err
}

func (s *Schedule) Next(t time.Time) time.Time {
	expr := (*cronexpr.Expression)(s)
	return expr.Next(t)
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
