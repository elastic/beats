package rpm

import "time"

// RPMDate is the Time format used by rpm tools.
const RPMDate = "Mon Jan _2 15:04:05 2006"

// Time provides formatting for time.Time as is typically used by tools in the
// RPM ecosystem.
type Time time.Time

func (c Time) String() string {
	return time.Time(c).UTC().Format(RPMDate)
}
