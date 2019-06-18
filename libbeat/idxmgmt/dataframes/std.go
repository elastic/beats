package dataframes

import (
	"github.com/elastic/beats/libbeat/logp"
)

type stdSupport struct {
	log       *logp.Logger
	mode      Mode
	source    string
	dest      string
	timespan  string
	transform map[string]interface{}
}

func (s *stdSupport) Mode() Mode {
	return s.mode
}

func (s *stdSupport) Source() string {
	return s.source
}

func (s *stdSupport) Dest() string {
	return s.dest
}

func (s *stdSupport) Timespan() string {
	return s.timespan
}

func (s *stdSupport) Transform() map[string]interface{} {
	return s.transform
}

func NewStdSupport(
	log *logp.Logger,
	mode Mode,
	source string,
	dest string,
	timespan string,
	transform map[string]interface{},
) Supporter {
	return &stdSupport{
		log:       log,
		mode:      mode,
		source:    source,
		dest:      dest,
		timespan:  timespan,
		transform: transform,
	}
}
