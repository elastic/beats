package add_locale

import (
	"fmt"
	"strings"
	"time"

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
	Abbrevation TimezoneFormat = iota
	Offset
)

var timezoneFormats = map[TimezoneFormat]string{
	Abbrevation: "abbrevation",
	Offset:      "offset",
}

func (t TimezoneFormat) String() string {
	return timezoneFormats[t]
}

func init() {
	processors.RegisterPlugin("add_locale", newAddLocale)
}

func newAddLocale(c common.Config) (processors.Processor, error) {
	config := struct {
		Format string `config:"format"`
	}{
		Format: "offset",
	}

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
		return nil, fmt.Errorf("'%s' is not a valid format option for processor add_locale. Valid options are 'abbrevation' and 'offset'", config.Format)

	}
	return loc, nil
}

func (l addLocale) Run(event common.MapStr) (common.MapStr, error) {
	zone, offset := time.Now().Zone()
	format := l.Format(zone, offset)
	event.Put("beat.timezone", format)

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
	case Abbrevation:
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
	return "add_locale=" + l.TimezoneFormat.String()
}
