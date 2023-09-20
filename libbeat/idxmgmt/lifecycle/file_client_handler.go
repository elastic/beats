package lifecycle

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// FileClientHandler implements the Loader interface for writing to a file.
type FileClientHandler struct {
	client        FileClient
	info          beat.Info
	cfg           Config
	defaultPolicy mapstr.M
	name          string
}

// NewFileClientHandler initializes and returns a new FileClientHandler instance.
func NewFileClientHandler(c FileClient, info beat.Info, cfg LifecycleConfig) (*FileClientHandler, error) {
	// TODO: should return an error if both are enabled
	// default to ILM for file handler
	if cfg.DSL.Enabled && cfg.ILM.Enabled {
		return nil, errors.New("both ILM and DLM are enabled")
	}

	rawName := cfg.ILM.PolicyName
	if cfg.DSL.Enabled {
		rawName = cfg.DSL.PolicyName
	}

	name, err := ApplyStaticFmtstr(info, rawName)
	if err != nil {
		return nil, fmt.Errorf("error creating policy name: %w", err)
	}

	if cfg.DSL.Enabled {
		return &FileClientHandler{client: c, info: info, cfg: cfg.DSL, defaultPolicy: DefaultDSLPolicy, name: name}, nil
	}
	return &FileClientHandler{client: c, info: info, cfg: cfg.ILM, defaultPolicy: DefaultILMPolicy, name: name}, nil

}

func (h *FileClientHandler) CheckExists() bool {
	return h.cfg.CheckExists
}

func (h *FileClientHandler) Overwrite() bool {
	return h.cfg.Enabled
}

// CheckEnabled indicates whether or not ILM is supported for the configured mode and client version.
// If the connected ES instance is serverless, this will return false
func (h *FileClientHandler) CheckEnabled() (bool, error) {
	return checkILMEnabled(h.cfg.Enabled, h.client)
}

// CreatePolicy writes given policy to the configured file.
func (h *FileClientHandler) CreatePolicy(policy Policy) error {
	str := fmt.Sprintf("%s\n", policy.Body.StringToPrint())
	if err := h.client.Write("policy", policy.Name, str); err != nil {
		return fmt.Errorf("error printing policy : %w", err)
	}
	return nil
}

func (h *FileClientHandler) CreatePolicyFromConfig() error {
	// only applicable to testing
	if h.cfg.policyRaw != nil {
		return h.CreatePolicy(*h.cfg.policyRaw)
	}

	// default to ILM
	policy, err := createPolicy(h.cfg, h.info, DefaultILMPolicy)
	if err != nil {
		return fmt.Errorf("error creating ILM policy: %w", err)
	}

	err = h.CreatePolicy(policy)
	if err != nil {
		return fmt.Errorf("error writing policy: %w", err)
	}
	return nil
}

func (h *FileClientHandler) PolicyName() string {
	return h.name
}

// HasPolicy always returns false.
func (h *FileClientHandler) HasPolicy() (bool, error) {
	return false, nil
}
