package clock

import (
	"time"
)

type Clock interface {
	Now() time.Time
	Sleep(duration time.Duration)
}
