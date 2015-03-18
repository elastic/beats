package inputs

import (
	"errors"
	"fmt"
	"packetbeat/common"
	"strings"
)

// The InputPlugin interface needs to be implemented
// by all the input plugins.
type InputPlugin interface {
	Init(test_mode bool, events chan common.MapStr) error
	Run() error
	Stop() error
	Close() error
	IsAlive() bool
}

type Input int

const (
	SnifferInput Input = iota
	UdpjsonInput
	GoBeaconInput
)

var InputPluginNames = []string{
	"sniffer",
	"udpjson",
	"gobeacon",
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

// CloseAll calls the Close methods of all registered inputs.
// All inputs Close() methods are called even when there are
// errors. The error messages are concatenated together.
func (inputs InputsList) CloseAll() error {
	errs := []string{}
	for input, plugin := range inputs.inputs {
		err := plugin.Close()
		if err != nil {
			errs = append(errs, fmt.Sprintf("Closing %s failed: %v", input, err))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, " "))
	}
	return nil
}

func (inputs InputsList) AreAllAlive() bool {
	for _, plugin := range inputs.inputs {
		if !plugin.IsAlive() {
			return false
		}
	}
	return true
}

func init() {
	Inputs = InputsList{}
	Inputs.inputs = make(map[Input]InputPlugin)
}
