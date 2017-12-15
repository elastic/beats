package log

import (
	"time"
)

type config struct {
	Period time.Duration `config:"period"`
}

var defaultConfig = config{
	Period: 30 * time.Second,
}
