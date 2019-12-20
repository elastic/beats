package testing

import (
	"math/rand"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // Number of letter indices fitting in 63 bits
)

// RandomBytes returns a random byte slice with length n.
func RandomBytes(n int) []byte {
	src := rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return b
}

const (
	// MaxPriority contains the maximum priority value that a RFC5424 syslog message can have.
	MaxPriority = uint8(191)
	// MaxVersion contains the maximum version value that a RFC5424 syslog message can have.
	MaxVersion = uint16(999)
	// MaxRFC3339MicroTimestamp contains the maximum length RFC3339MICRO timestamp that a RFC5424 syslog message can have.
	MaxRFC3339MicroTimestamp = "2018-12-31T23:59:59.999999-23:59"
)

var (
	// MaxHostname is a maximum length hostname that a RFC5424 syslog message can have.
	MaxHostname = RandomBytes(255)
	// MaxAppname is a maximum length app-name that a RFC5424 syslog message can have.
	MaxAppname = RandomBytes(48)
	// MaxProcID is a maximum length app-name that a RFC5424 syslog message can have.
	MaxProcID = RandomBytes(128)
	// MaxMsgID is a maximum length app-name that a RFC5424 syslog message can have.
	MaxMsgID = RandomBytes(32)
	// MaxMessage is a maximum length message that a RFC5424 syslog message can contain when all other fields are at their maximum length.
	MaxMessage = RandomBytes(7681)
)

// RightPad pads a string with spaces until the given limit, or it cuts the string to the given limit.
func RightPad(str string, limit int) string {
	str = str + strings.Repeat(" ", limit)
	return str[:limit]
}

// StringAddress returns the address of the input string.
func StringAddress(str string) *string {
	return &str
}

// Uint8Address returns the address of the input uint8.
func Uint8Address(x uint8) *uint8 {
	return &x
}

// TimeParse parses a time string, for the given layout, into a pointer to a time.Time instance.
func TimeParse(layout, value string) *time.Time {
	t, _ := time.Parse(layout, value)
	return &t
}

// YearTime returns a time.Time of the given month, day, hour, minutes, and seconds for the current year (in UTC).
func YearTime(mm, dd, hh, min, ss int) time.Time {
	return time.Date(time.Now().Year(), time.Month(mm), dd, hh, min, ss, 0, time.UTC)
}
