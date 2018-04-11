package syslog

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSeverity(t *testing.T) {
	e := newEvent()
	e.SetPriority([]byte("13"))
	assert.Equal(t, 5, e.Severity())
}

func TestFacility(t *testing.T) {
	e := newEvent()
	e.SetPriority([]byte("13"))
	assert.Equal(t, 1, e.Facility())
}

func TestHasPriority(t *testing.T) {
	e := newEvent()
	e.SetPriority([]byte("13"))
	assert.True(t, e.HasPriority())
	assert.Equal(t, 13, e.Priority())
	assert.Equal(t, 5, e.Severity())
	assert.Equal(t, 1, e.Facility())
}

func TestNoPrioritySet(t *testing.T) {
	e := newEvent()
	assert.False(t, e.HasPriority())
	assert.Equal(t, -1, e.Priority())
	assert.Equal(t, -1, e.Severity())
	assert.Equal(t, -1, e.Facility())
}

func TestHasPid(t *testing.T) {
	e := newEvent()
	assert.False(t, e.HasPid())
	e.SetPid([]byte(strconv.Itoa(20)))
	assert.True(t, e.HasPid())
}

func TestDateParsing(t *testing.T) {
	now := time.Now()
	e := newEvent()
	e.SetDay(itb(now.Day()))
	e.SetMonth([]byte(now.Month().String()))
	e.SetHour(itb(now.Hour()))
	e.SetMinute(itb(now.Minute()))
	e.SetSecond(itb(now.Second()))
	e.SetNanosecond(itb(now.Nanosecond()))
	new := e.Timestamp(time.Local)
	assert.Equal(t, now.UTC(), new)
}

func TestIsValid(t *testing.T) {
	e := newEvent()
	assert.False(t, e.IsValid())

	now := time.Now()

	e.SetDay(itb(now.Day()))
	assert.False(t, e.IsValid())

	e.SetMonth([]byte(now.Month().String()))
	assert.False(t, e.IsValid())

	e.SetHour(itb(now.Hour()))
	assert.False(t, e.IsValid())

	e.SetMinute(itb(now.Minute()))
	assert.False(t, e.IsValid())

	e.SetSecond(itb(now.Second()))
	assert.False(t, e.IsValid())

	e.SetMessage([]byte("hello world"))
	assert.True(t, e.IsValid())
}

func itb(i int) []byte {
	if i < 10 {
		return []byte(fmt.Sprintf("0%d", i))
	}
	return []byte(strconv.Itoa(i))
}
