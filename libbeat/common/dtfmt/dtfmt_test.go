// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package dtfmt

import (
	"fmt"
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
		{mkTime(8, 5, 24, 0), "K:m:s a", "8:5:24 AM"},
		{mkTime(8, 5, 24, 0), "KK:mm:ss aa", "08:05:24 AM"},
		{mkTime(20, 5, 24, 0), "K:m:s a", "8:5:24 PM"},
		{mkTime(20, 5, 24, 0), "KK:mm:ss aa", "08:05:24 PM"},
		{mkTime(8, 5, 24, 0), "h:m:s a", "9:5:24 AM"},
		{mkTime(8, 5, 24, 0), "hh:mm:ss aa", "09:05:24 AM"},
		{mkTime(20, 5, 24, 0), "h:m:s a", "9:5:24 PM"},
		{mkTime(20, 5, 24, 0), "hh:mm:ss aa", "09:05:24 PM"},
		{mkTime(8, 5, 24, 0), "H:m:s a", "8:5:24 AM"},
		{mkTime(8, 5, 24, 0), "HH:mm:ss aa", "08:05:24 AM"},
		{mkTime(20, 5, 24, 0), "H:m:s a", "20:5:24 PM"},
		{mkTime(20, 5, 24, 0), "HH:mm:ss aa", "20:05:24 PM"},
		{mkTime(8, 5, 24, 0), "k:m:s a", "9:5:24 AM"},
		{mkTime(8, 5, 24, 0), "kk:mm:ss aa", "09:05:24 AM"},
		{mkTime(20, 5, 24, 0), "k:m:s a", "21:5:24 PM"},
		{mkTime(20, 5, 24, 0), "kk:mm:ss aa", "21:05:24 PM"},
		{mkTime(1, 2, 3, 123), "S", "1"},
		{mkTime(1, 2, 3, 123), "SS", "12"},
		{mkTime(1, 2, 3, 123), "SSS", "123"},
		{mkTime(1, 2, 3, 123), "SSSS", "1230"},

		// literals
		{time.Now(), "--=++,_!/?\\[]{}@#$%^&*()", "--=++,_!/?\\[]{}@#$%^&*()"},
		{time.Now(), "'plain text'", "plain text"},
		{time.Now(), "'plain' 'text'", "plain text"},
		{time.Now(), "'plain' '' 'text'", "plain ' text"},
		{time.Now(), "'plain '' text'", "plain ' text"},

		// beats timestamp
		{mkDateTime(2017, 1, 2, 4, 6, 7, 123),
			"yyyy-MM-dd'T'HH:mm:ss.SSS'Z'",
			"2017-01-02T04:06:07.123Z"},
	}

	for i, test := range tests {
		name := fmt.Sprintf("run (%v): %v -> %v", i, test.pattern, test.expected)
		t.Run(name, func(t *testing.T) {
			actual, err := Format(test.time, test.pattern)
			if err != nil {
				t.Error(err)
				return
			}

			assert.Equal(t, test.expected, actual)
		})
	}
}

func mkDate(y, m, d int) time.Time {
	return mkDateTime(y, m, d, 0, 0, 0, 0)
}

func mkTime(h, m, s, S int) time.Time {
	return mkDateTime(2000, 1, 1, h, m, s, S)
}

func mkDateTime(y, M, d, h, m, s, S int) time.Time {
	return time.Date(y, time.Month(M), d, h, m, s, S*1000000, time.UTC)
}
