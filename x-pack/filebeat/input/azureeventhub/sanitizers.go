package azureeventhub

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ----------------------------------------------------------------------------
// Sanitizer API
// ----------------------------------------------------------------------------

type SanitizerSpec struct {
	Type string                 `config:"type"`
	Spec map[string]interface{} `config:"spec"`
}
type Sanitizer interface {
	Sanitize(jsonByte []byte) []byte
	Init() error
}

func newSanitizer(spec SanitizerSpec) (Sanitizer, error) {
	var s Sanitizer

	switch spec.Type {
	case "new_lines":
		s = &newLinesSanitizer{}
	case "single_quotes":
		s = &singleQuotesSanitizer{}
	case "regex":
		s = &regexpSanitizer{spec: spec.Spec}
	default:
		return nil, fmt.Errorf("unknown sanitizer type: %s", spec.Type)
	}

	// Initialize the sanitizer with the provided spec.
	err := s.Init()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func newSanitizers(specs []SanitizerSpec) ([]Sanitizer, error) {
	var sanitizers []Sanitizer

	for _, spec := range specs {
		sanitizer, err := newSanitizer(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to build sanitizer: %w", err)
		}

		sanitizers = append(sanitizers, sanitizer)
	}

	return sanitizers, nil
}

// ----------------------------------------------------------------------------
// New line sanitizer
// ----------------------------------------------------------------------------

type newLinesSanitizer struct{}

func (s *newLinesSanitizer) Sanitize(jsonByte []byte) []byte {
	return bytes.ReplaceAll(jsonByte, []byte("\n"), []byte{})
}

func (s *newLinesSanitizer) Init() error {
	return nil
}

// ----------------------------------------------------------------------------
// Single quote sanitizer
// ----------------------------------------------------------------------------

type singleQuotesSanitizer struct{}

func (s *singleQuotesSanitizer) Sanitize(jsonByte []byte) []byte {
	var result bytes.Buffer
	var prevChar byte

	inDoubleQuotes := false

	for _, r := range jsonByte {
		if r == '"' && prevChar != '\\' {
			inDoubleQuotes = !inDoubleQuotes
		}

		if r == '\'' && !inDoubleQuotes {
			result.WriteRune('"')
		} else {
			result.WriteByte(r)
		}
		prevChar = r
	}

	return result.Bytes()
}

func (s *singleQuotesSanitizer) Init() error {
	return nil
}

// ----------------------------------------------------------------------------
// Regular expression sanitizer
// ----------------------------------------------------------------------------

type regexpSanitizer struct {
	spec map[string]interface{}
	re   *regexp.Regexp
}

func (s *regexpSanitizer) Sanitize(jsonByte []byte) []byte {
	if s.re == nil {
		return jsonByte
	}

	// Remove any leading/trailing whitespace
	input := strings.TrimSpace(string(jsonByte))

	// Regular expression to match array contents that are not valid JSON
	// re := regexp.MustCompile(`\[\s*([^[\]{},\s]+(?:\s+[^[\]{},\s]+)*)\s*\]`)
	// re := regexp.MustCompile(`\[\s*([^\[\]{},\s]+(?:\s+[^\[\]{},\s]+)*)\s*\]`)

	// Replace invalid array contents with a string placeholder
	sanitized := s.re.ReplaceAllStringFunc(input, func(match string) string {
		return fmt.Sprintf("[\"%s\"]", strings.TrimSpace(match[1:len(match)-1]))
	})

	return []byte(sanitized)
}

func (s *regexpSanitizer) Init() error {
	if s.spec == nil {
		return errors.New("missing sanitizer spec")
	}

	if _, ok := s.spec["pattern"]; !ok {
		return errors.New("missing regex pattern")
	}

	if _, ok := s.spec["pattern"].(string); !ok {
		return errors.New("regex pattern must be a string")
	}

	re, err := regexp.Compile(s.spec["pattern"].(string))
	if err != nil {
		return err
	}

	s.re = re

	return nil
}
