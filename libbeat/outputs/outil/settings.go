package outil

import "strings"

// Settings configures how BuildSelectorFromConfig creates a Selector from
// a given configuration object.
type Settings struct {
	// single selector key and default option keyword
	Key string

	// multi-selector key in config
	MultiKey string

	// if enabled a selector `key` in config will be generated, if `key` is present
	EnableSingleOnly bool

	// Fail building selector if `key` and `multiKey` are missing
	FailEmpty bool

	// Case configures the case-sensitivity of generated strings.
	Case SelectorCase
}

type SelectorCase uint8

const (
	SelectorKeepCase SelectorCase = iota
	SelectorLowerCase
	SelectorUpperCase
)

func (s Settings) WithKey(key string) Settings {
	s.Key = key
	return s
}

func (s Settings) WithMultiKey(key string) Settings {
	s.MultiKey = key
	return s
}

func (s Settings) WithEnableSingleOnly(b bool) Settings {
	s.EnableSingleOnly = b
	return s
}

func (s Settings) WithFailEmpty(b bool) Settings {
	s.FailEmpty = b
	return s
}

func (s Settings) WithSelectorCase(c SelectorCase) Settings {
	s.Case = c
	return s
}

func (selCase SelectorCase) apply(in string) string {
	switch selCase {
	case SelectorLowerCase:
		return strings.ToLower(in)
	case SelectorUpperCase:
		return strings.ToUpper(in)
	default:
		return in
	}
}
