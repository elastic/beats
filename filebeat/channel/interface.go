package channel

import "github.com/elastic/beats/filebeat/util"

// Outleter is the outlet for a prospector
type Outleter interface {
	SetSignal(signal <-chan struct{})
	OnEventSignal(data *util.Data) bool
	OnEvent(data *util.Data) bool
	Copy() Outleter
}
