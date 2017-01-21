package dtfmt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		time     time.Time
		pattern  string
		expected string
	}{
		// year.month.day of month
		{mkDate(6, 8, 1), "y.M.d", "6.8.1"},
		{mkDate(2006, 8, 1), "y.M.d", "2006.8.1"},
		{mkDate(2006, 8, 1), "yy.MM.dd", "06.08.01"},
		{mkDate(6, 8, 1), "yy.MM.dd", "06.08.01"},
		{mkDate(2006, 8, 1), "yyy.MMM.dd", "2006.Aug.01"},
		{mkDate(2006, 8, 1), "yyyy.MMMM.d", "2006.August.1"},
		{mkDate(2006, 8, 1), "yyyyyy.MM.ddd", "002006.08.001"},

		// year of era.month.day
		{mkDate(6, 8, 1), "Y.M.d", "6.8.1"},
		{mkDate(2006, 8, 1), "Y.M.d", "2006.8.1"},
		{mkDate(2006, 8, 1), "YY.MM.dd", "06.08.01"},
		{mkDate(6, 8, 1), "YY.MM.dd", "06.08.01"},
		{mkDate(2006, 8, 1), "YYY.MMM.dd", "2006.Aug.01"},
		{mkDate(2006, 8, 1), "YYYY.MMMM.d", "2006.August.1"},
		{mkDate(2006, 8, 1), "YYYYYY.MM.ddd", "002006.08.001"},

		// week year + week of year + day of week
		{mkDate(2015, 1, 1), "xx.ww.e", "15.01.4"},
		{mkDate(2014, 12, 31), "xx.ww.e", "15.01.3"},
		{mkDate(2015, 1, 1), "xx.w.E", "15.1.Thu"},
		{mkDate(2014, 12, 31), "xx.w.E", "15.1.Wed"},
		{mkDate(2015, 1, 1), "xx.w.EEEE", "15.1.Thursday"},
		{mkDate(2014, 12, 31), "xx.w.EEEE", "15.1.Wednesday"},
		{mkDate(2015, 1, 1), "xxxx.ww", "2015.01"},
		{mkDate(2014, 12, 31), "xxxx.ww", "2015.01"},
		{mkDate(2015, 1, 1), "xxxx.ww.e", "2015.01.4"},
		{mkDate(2014, 12, 31), "xxxx.ww.e", "2015.01.3"},
		{mkDate(2015, 1, 1), "xxxx.w.E", "2015.1.Thu"},
		{mkDate(2014, 12, 31), "xxxx.w.E", "2015.1.Wed"},
		{mkDate(2015, 1, 1), "xxxx.w.EEEE", "2015.1.Thursday"},
		{mkDate(2014, 12, 31), "xxxx.w.EEEE", "2015.1.Wednesday"},

		// time
		{mkTime(8, 5, 24), "K:m:s a", "8:5:24 AM"},
		{mkTime(8, 5, 24), "KK:mm:ss aa", "08:05:24 AM"},
		{mkTime(20, 5, 24), "K:m:s a", "8:5:24 PM"},
		{mkTime(20, 5, 24), "KK:mm:ss aa", "08:05:24 PM"},
		{mkTime(8, 5, 24), "h:m:s a", "9:5:24 AM"},
		{mkTime(8, 5, 24), "hh:mm:ss aa", "09:05:24 AM"},
		{mkTime(20, 5, 24), "h:m:s a", "9:5:24 PM"},
		{mkTime(20, 5, 24), "hh:mm:ss aa", "09:05:24 PM"},
		{mkTime(8, 5, 24), "H:m:s a", "8:5:24 AM"},
		{mkTime(8, 5, 24), "HH:mm:ss aa", "08:05:24 AM"},
		{mkTime(20, 5, 24), "H:m:s a", "20:5:24 PM"},
		{mkTime(20, 5, 24), "HH:mm:ss aa", "20:05:24 PM"},
		{mkTime(8, 5, 24), "k:m:s a", "9:5:24 AM"},
		{mkTime(8, 5, 24), "kk:mm:ss aa", "09:05:24 AM"},
		{mkTime(20, 5, 24), "k:m:s a", "21:5:24 PM"},
		{mkTime(20, 5, 24), "kk:mm:ss aa", "21:05:24 PM"},

		// literals
		{time.Now(), "--=++,_!/?\\[]{}@#$%^&*()", "--=++,_!/?\\[]{}@#$%^&*()"},
		{time.Now(), "'plain text'", "plain text"},
		{time.Now(), "'plain' 'text'", "plain text"},
		{time.Now(), "'plain' '' 'text'", "plain ' text"},
		{time.Now(), "'plain '' text'", "plain ' text"},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v -> %v", i, test.pattern, test.expected)

		actual, err := Format(test.time, test.pattern)
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.expected, actual)
	}
}

func mkDate(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.Local)
}

func mkTime(h, m, s int) time.Time {
	return time.Date(2000, 1, 1, h, m, s, 0, time.Local)
}
