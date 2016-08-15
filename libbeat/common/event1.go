package common

import (
	"fmt"
	"time"
)

type EventField struct {
	integer   *int
	float     *float64
	boolean   *bool
	str       *string
	timestamp *time.Time
	dict      *Dict
}

type Event map[string]EventField
type Dict map[string]EventField

func Int(i int) EventField {

	return EventField{integer: &i}
}

func Str(s string) EventField {
	return EventField{str: &s}
}

func Timestamp(t time.Time) EventField {
	return EventField{timestamp: &t}
}

func Nested(d Dict) EventField {
	return EventField{dict: &d}
}

func (ev EventField) Int() int {
	return *ev.integer
}

func (ev EventField) Float() float64 {
	return *ev.float
}

func (ev EventField) Bool() bool {
	return *ev.boolean
}

func (ev EventField) Str() string {
	return *ev.str
}

func (ev EventField) Timestamp() time.Time {
	return *ev.timestamp
}

func (ev EventField) Nested() Dict {
	return *ev.dict
}

func (ev EventField) String() string {
	if ev.integer != nil {
		return fmt.Sprintf("%d", ev.Int())
	}
	if ev.float != nil {
		return fmt.Sprintf("%.2f", ev.Float())
	}
	if ev.boolean != nil {
		return fmt.Sprintf("%t", ev.Bool())
	}
	if ev.str != nil {
		return ev.Str()
	}
	if ev.timestamp != nil {
		return fmt.Sprintf("%v", ev.Timestamp())
	}
	if ev.dict != nil {
		return fmt.Sprintf("%v", ev.Nested())
	}
	return "<nil>"
}
