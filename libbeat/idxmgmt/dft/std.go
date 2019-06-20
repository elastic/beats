package dft

import (
	"github.com/elastic/beats/libbeat/logp"
)

type dfSupport struct {
	log  *logp.Logger
	mode Mode

	transform DataFrameTransform
}

type stdManager struct {
	mode   Mode
	client ClientHandler
	dft    DataFrameTransform

	cached             bool
	cachedEnabledValue bool
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
	return m.client.EnsureDataFrames(m.dft)
}

func (s *dfSupport) Manager(h ClientHandler) Manager {
	return &stdManager{
		mode:   s.mode,
		client: h,
		dft:    s.transform,
	}
}

func (s *dfSupport) Mode() Mode {
	return s.mode
}

func (s *dfSupport) Transform() DataFrameTransform {
	return s.transform
}

func NewStdSupport(
	log *logp.Logger,
	mode Mode,
	transform DataFrameTransform,
) Supporter {
	return &dfSupport{
		log:       log,
		mode:      mode,
		transform: transform,
	}
}
