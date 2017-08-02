package add_locale

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
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
	processors.RegisterPlugin("add_locale", newAddLocale)
}

func newAddLocale(c *common.Config) (processors.Processor, error) {
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
	event.PutValue("beat.timezone", format)
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
