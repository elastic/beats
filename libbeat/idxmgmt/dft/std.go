package dft

import (
	"github.com/elastic/beats/libbeat/logp"
)

type dfSupport struct {
	log  *logp.Logger
	mode Mode

	transforms []*DataFrameTransform
}

type stdManager struct {
	mode   Mode
	client ClientHandler
	dft    []*DataFrameTransform

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
	return m.client.EnsureDataFrameTransforms(m.dft)
}

func (s *dfSupport) Manager(h ClientHandler) Manager {
	return &stdManager{
		mode:   s.mode,
		client: h,
		dft:    s.transforms,
	}
}

func (s *dfSupport) Mode() Mode {
	return s.mode
}

func (s *dfSupport) Transforms() []*DataFrameTransform {
	return s.transforms
}

func NewStdSupport(
	log *logp.Logger,
	mode Mode,
	transforms []*DataFrameTransform,
) Supporter {
	return &dfSupport{
		log:        log,
		mode:       mode,
		transforms: transforms,
	}
}
