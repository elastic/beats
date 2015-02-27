package inputs

import "packetbeat/common"

// The InputPlugin interface needs to be implemented
// by the input plugins.
type InputPlugin interface {
	Init(test_mode bool, events chan common.MapStr) error
	Run() error
	Stop() error
	Close() error
}

type Input int

const (
	SnifferInput Input = iota
	UdpjsonInput
)

var InputPluginNames = []string{
	"sniffer",
	"udpjson",
}

func (input Input) String() string {
	if int(input) >= len(InputPluginNames) {
		return "impossible"
	}
	return InputPluginNames[input]
}

// Check if the input name is in a list of names.
func (input Input) IsInList(lst []string) bool {
	for _, name := range lst {
		if name == input.String() {
			return true
		}
	}
	return false
}

// Contains a list of the available input plugins.
type InputsList struct {
	inputs map[Input]InputPlugin
}

func (inputs InputsList) Get(input Input) InputPlugin {
	ret, exists := inputs.inputs[input]
	if !exists {
		return nil
	}
	return ret
}

var Inputs InputsList

func (inputs InputsList) Register(input Input, plugin InputPlugin) {
	inputs.inputs[input] = plugin
}

func (inputs InputsList) Registered() map[Input]InputPlugin {
	return inputs.inputs
}

// StopAll calls the Stop methods of all registered inputs
func (inputs InputsList) StopAll() {
	for _, plugin := range inputs.inputs {
		plugin.Stop()
	}
}

func init() {
	Inputs = InputsList{}
	Inputs.inputs = make(map[Input]InputPlugin)
}
