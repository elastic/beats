package converter

import (
	"fmt"

	"golang.org/x/text/transform"

	"github.com/elastic/beats/filebeat/reader/encode/encoding"
)

type Converter interface {
	Collect(out []byte) (int, error)
	Convert(in, out []byte) (int, int, error)
	MsgSize(symlen []uint8, size int) (int, []uint8, error)
	GetSymLen() []uint8
}

type ConverterFactory = func(t transform.Transformer, size int) (Converter, error)

var registry = make(map[string]ConverterFactory)

func register(name string, factory ConverterFactory) error {
	if name == "" {
		return fmt.Errorf("Error registering Converter factory: name cannot be empty")
	}
	if _, exists := encoding.FindEncoding(name); !exists && name != "default" {
		return fmt.Errorf("Error registering Converter factory for encoding '%v': no such encoding", name)
	}
	if factory == nil {
		return fmt.Errorf("Error registering Converter factory for '%v': callbacks cannot be empty", name)
	}
	if _, exists := registry[name]; exists {
		return fmt.Errorf("Error registering Converter factory for '%v': already registered", name)
	}

	registry[name] = factory
	return nil
}

func GetFactory(name string) (ConverterFactory, error) {
	if _, exists := encoding.FindEncoding(name); !exists {
		return nil, fmt.Errorf("Unknown encoding configured", name)
	}

	if _, exists := registry[name]; !exists {
		return registry["default"], nil
	}
	return registry[name], nil
}
