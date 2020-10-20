package transforms

import "time"

var (
	predefinedLayouts = map[string]string{
		"ANSIC":       time.ANSIC,
		"UnixDate":    time.UnixDate,
		"RubyDate":    time.RubyDate,
		"RFC822":      time.RFC822,
		"RFC822Z":     time.RFC822Z,
		"RFC850":      time.RFC850,
		"RFC1123":     time.RFC1123,
		"RFC1123Z":    time.RFC1123Z,
		"RFC3339":     time.RFC3339,
		"RFC3339Nano": time.RFC3339Nano,
		"Kitchen":     time.Kitchen,
	}
)

func formatDate(date time.Time, layout string, tz ...string) string {
	if found := predefinedLayouts[layout]; found != "" {
		layout = found
	} else {
		layout = time.RFC3339
	}

	if len(tz) > 0 {
		if loc, err := time.LoadLocation(tz[0]); err == nil {
			date = date.In(loc)
		} else {
			date = date.UTC()
		}
	} else {
		date = date.UTC()
	}

	return date.Format(layout)
}

func parseDate(date, layout string) time.Time {
	if found := predefinedLayouts[layout]; found != "" {
		layout = found
	} else {
		layout = time.RFC3339
	}

	t, err := time.Parse(layout, date)
	if err != nil {
		return time.Time{}
	}

	return t
}

func now(add ...time.Duration) time.Time {
	now := time.Now()
	if len(add) == 0 {
		return now
	}
	return now.Add(add[0])
}

func getRFC5988Link(links, rel string) string {
	return ""
}
