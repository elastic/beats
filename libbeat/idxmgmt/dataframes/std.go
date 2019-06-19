package dataframes

import (
	"fmt"

	"github.com/elastic/beats/libbeat/logp"
)

type dfSupport struct {
	log       *logp.Logger
	mode      Mode
	source    string
	dest      string
	timespan  string
	transform map[string]interface{}
}

type stdManager struct {
	mode   Mode
	client ClientHandler
}

func (m *stdManager) Enabled() (bool, error) {
	if m.mode == ModeDisabled {
		return false, nil
	}

	enabled, err := m.client.CheckDataFramesEnabled(m.mode)
	if err != nil {
		return enabled, err
	}

	if !enabled && m.mode == ModeEnabled {
		return false, ErrESDFDisabled
	}

	return enabled, nil
}

func (m *stdManager) EnsureDataframes() error {
	fmt.Printf("ENSURING DATA FRAMES")
	return m.EnsureDataframes()
}

func (s *dfSupport) Manager(h ClientHandler) Manager {
	return &stdManager{
		mode:   s.mode,
		client: h,
	}
}

func (s *dfSupport) Mode() Mode {
	return s.mode
}

func (s *dfSupport) Source() string {
	return s.source
}

func (s *dfSupport) Dest() string {
	return s.dest
}

func (s *dfSupport) Timespan() string {
	return s.timespan
}

func (s *dfSupport) Transform() map[string]interface{} {
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
	return &dfSupport{
		log:       log,
		mode:      mode,
		source:    source,
		dest:      dest,
		timespan:  timespan,
		transform: transform,
	}
}
