package lifecycle

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// ESClientHandler implements the Loader interface for talking to ES.
type ESClientHandler struct {
	client        ESClient
	info          beat.Info
	cfg           Config
	defaultPolicy mapstr.M
	putPath       string
	name          string
}

// NewESClientHandler initializes and returns an ESClientHandler
func NewESClientHandler(c ESClient, info beat.Info, cfg LifecycleConfig) (*ESClientHandler, error) {
	policyName := cfg.ILM.PolicyName
	if c.IsServerless() {
		policyName = cfg.DSL.PolicyName
	}

	// trying to protect against config confusion;
	// it's possible that the "wrong" lifecycle got enabled somehow,
	// this is a last-ditch effort to fix things
	if (!cfg.DSL.Enabled && cfg.ILM.Enabled && c.IsServerless()) || (!cfg.ILM.Enabled && cfg.DSL.Enabled && !c.IsServerless()) {
		log := logp.L()
		log.Warnf("lifecycle config setup does not the type of ES we're connected to. serverless=%b, yet config ILM=%b DSL=%b",
			c.IsServerless(), cfg.ILM.Enabled, cfg.DSL.Enabled)
		// assume we want some kind of lifecycle management
		if c.IsServerless() {
			cfg.DSL.Enabled = true
		} else {
			cfg.ILM.Enabled = true
		}
	}

	name, err := ApplyStaticFmtstr(info, policyName)
	if err != nil {
		return nil, fmt.Errorf("error applying format string to policy name: %w", err)
	}

	if name == "" {
		return nil, errors.New("could not generate usable policy name from config. Check setup.*.policy_name fields")
	}

	if c.IsServerless() {
		path := fmt.Sprintf("/_data_stream/%s/_lifecycle", name)
		return &ESClientHandler{client: c, info: info, cfg: cfg.DSL, defaultPolicy: DefaultDSLPolicy, name: name, putPath: path}, nil
	}

	path := fmt.Sprintf("%s/%s", esILMPath, name)
	return &ESClientHandler{client: c, info: info, cfg: cfg.ILM, defaultPolicy: DefaultILMPolicy, name: name, putPath: path}, nil
}

// CheckExists returns the value of the check_exists config flag
func (h *ESClientHandler) CheckExists() bool {
	return h.cfg.CheckExists
}

// returns the value of the overwrite config flag
func (h *ESClientHandler) Overwrite() bool {
	return h.cfg.Overwrite
}

// CheckEnabled indicates whether or not ILM is supported for the configured mode and ES instance.
func (h *ESClientHandler) CheckEnabled() (bool, error) {
	return checkILMEnabled(h.cfg.Enabled, h.client)
}

// HasPolicy queries Elasticsearch to see if policy with given name exists.
func (h *ESClientHandler) HasPolicy() (bool, error) {
	status, b, err := h.client.Request("GET", h.putPath, "", nil, nil)
	if err != nil && status != 404 {
		return false, wrapErrf(err, ErrRequestFailed,
			"failed to check for policy name '%v': (status=%v) %s", h.name, status, b)
	}
	return status == 200, nil
}

// CreatePolicyFromConfig creates a DSL policy from a raw setup config for the beat
func (h *ESClientHandler) CreatePolicyFromConfig() error {
	// check overwrite before we do this
	// normally other upstream components do this check,
	// but might as well do it here
	if !h.cfg.Overwrite {
		found, err := h.HasPolicy()
		if err != nil {
			return fmt.Errorf("error looking for existing policy: %w", err)
		}
		// maintain old behavior, don't return an error
		if found {
			return nil
		}
	}
	// only applicable to testing
	if h.cfg.policyRaw != nil {
		return h.putPolicyToES(h.putPath, *h.cfg.policyRaw)
	}

	err := h.createAndPutPolicy(h.cfg, h.info, h.defaultPolicy)
	if err != nil {
		return fmt.Errorf("error creating policy from config: %w", err)
	}
	return nil
}

// PolicyName returns the policy name
func (h *ESClientHandler) PolicyName() string {
	return h.name
}

// creates a policy from config, then performs the PUT request to ES
func (h *ESClientHandler) createAndPutPolicy(cfg Config, info beat.Info, defaultPolicy mapstr.M) error {
	policy, err := createPolicy(cfg, info, defaultPolicy)
	if err != nil {
		return fmt.Errorf("error creating lifetime policy: %w", err)
	}
	err = h.putPolicyToES(h.putPath, policy)
	if err != nil {
		return fmt.Errorf("error submitting policy: %w", err)
	}
	return nil
}

// performs the PUT operation to create a policy
func (h *ESClientHandler) putPolicyToES(path string, policy Policy) error {
	retCode, resp, err := h.client.Request("PUT", path, "", nil, policy.Body)
	if retCode > 300 {
		return fmt.Errorf("error creating lifecycle policy: got %d from elasticsearch: %s", retCode, resp)
	}
	if err != nil {
		return fmt.Errorf("error in lifecycle PUT request: %w", err)
	}
	return nil
}
