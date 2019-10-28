package quantize

import "time"

func Quantize(t time.Time, period time.Duration) (start time.Time, end time.Time) {
	periodUnix := time.Unix(int64(period/time.Second), 0).Unix()
	start = time.Unix((t.Unix()/periodUnix)*periodUnix, 0)
	end = start.Add(period)
	return start, end
}
