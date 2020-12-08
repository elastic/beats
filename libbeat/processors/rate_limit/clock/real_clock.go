package clock

import "time"

type RealClock struct{}

func (r RealClock) Now() time.Time {
	return time.Now()
}

func (r RealClock) Sleep(d time.Duration) {
	time.Sleep(d)
}
