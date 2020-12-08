package clock

import "time"

type FakeClock struct {
	now time.Time
}

func (f *FakeClock) Now() time.Time {
	return f.now
}

func (f *FakeClock) SetNow(now time.Time) {
	f.now = now
}

func (f *FakeClock) Sleep(d time.Duration) {
	f.now = f.now.Add(d)
}
