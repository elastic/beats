package actions

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type addLocale struct {
	TimezoneFormat TimezoneFormat
}

type TimezoneFormat int

const (
	Abbrevation TimezoneFormat = iota
	Offset
)

var timezoneFormats = [...]string{
	"abbrevation",
	"offset",
}

func (t TimezoneFormat) String() string {
	return timezoneFormats[t]
}

func init() {
	processors.RegisterPlugin("add_locale", newAddLocale)
}

func newAddLocale(c common.Config) (processors.Processor, error) {
	config := struct {
		Format string `config:"format" validate:"required"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the include_fields configuration: %s", err)
	}

	loc := addLocale{}

	switch strings.ToLower(config.Format) {
	case "abbrevation":
		loc.TimezoneFormat = Abbrevation
	case "offset":
		loc.TimezoneFormat = Offset
	default:
		return nil, errors.New("given format is not valid")

	}
	return loc, nil
}

func (l addLocale) Run(event common.MapStr) (common.MapStr, error) {
	format := l.Format()
	event.Put("beat.timezone", format)

	return event, nil
}

func (l addLocale) Format() string {
	tm := time.Now()
	var fmt string
	switch l.TimezoneFormat {
	case Abbrevation:
		fmt, _ = tm.Zone()
	case Offset:
		//loc, _ := time.LoadLocation("America/Belize")
		_, offset := tm.Zone()

		offset = offset / 3600

		if offset < 0 {
			offset = offset - (offset * 2)
			fmt += "-"
		} else {
			fmt += "+"
		}

		if offset < 10 {
			fmt += "0"
		}

		fmt += strconv.Itoa(offset)
		fmt += ":00"
	}
	return fmt
}

func (l addLocale) String() string {
	return "add_locale=" + l.TimezoneFormat.String()
}
