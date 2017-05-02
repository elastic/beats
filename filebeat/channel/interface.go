package channel

import "github.com/elastic/beats/filebeat/input"

// Outleter is the outlet for a prospector
type Outleter interface {
	SetSignal(signal <-chan struct{})
	OnEventSignal(event *input.Data) bool
	OnEvent(event *input.Data) bool
	Copy() Outleter
}
