package transforms

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const logName = "httpjson.transforms"

// Config represents the list of transforms.
type Config []*common.Config

type Transforms struct {
	List []Transform
	log  *logp.Logger
}

type Transform interface {
	Run(*Transformable) (*Transformable, error)
	String() string
}

type TargetType int

const (
	TargetBody TargetType = iota
	TargetCursor
	TargetHeaders
	TargetURLValue
	TargetURLParams
)

type ErrInvalidTarget struct {
	target string
}

func (err ErrInvalidTarget) Error() string {
	return fmt.Sprintf("invalid target %q", err.target)
}

type Transformable struct {
	Headers      http.Header
	Body         common.MapStr
	URL          url.URL
	Cursor       common.MapStr
	LastEvent    common.MapStr
	LastResponse common.MapStr
}

func NewEmptyTransformable() *Transformable {
	return &Transformable{
		Headers:      make(http.Header),
		Body:         make(common.MapStr),
		Cursor:       make(common.MapStr),
		LastEvent:    make(common.MapStr),
		LastResponse: make(common.MapStr),
	}
}

type TargetInfo struct {
	Type TargetType
	Name string
}

// NewList creates a new empty transform list.
// Additional processors can be added to the List field.
func NewList(log *logp.Logger) *Transforms {
	if log == nil {
		log = logp.NewLogger(logName)
	}
	return &Transforms{log: log}
}

// New creates a list of transforms from a list of free user configurations.
func New(config Config, namespace string) (*Transforms, error) {
	trans := NewList(nil)

	for _, tfConfig := range config {
		if len(tfConfig.GetFields()) != 1 {
			return nil, errors.Errorf(
				"each transform must have exactly one action, but found %d actions (%v)",
				len(tfConfig.GetFields()),
				strings.Join(tfConfig.GetFields(), ","),
			)
		}

		actionName := tfConfig.GetFields()[0]
		cfg, err := tfConfig.Child(actionName, -1)
		if err != nil {
			return nil, err
		}

		constructor, found := registeredTransforms.get(namespace, actionName)
		if !found {
			return nil, errors.Errorf("the transform %s does not exist. Valid transforms: %s", actionName, registeredTransforms.String())
		}

		cfg.PrintDebugf("Configure transform '%v' with:", actionName)
		transform, err := constructor(cfg)
		if err != nil {
			return nil, err
		}

		trans.Add(transform)
	}

	if len(trans.List) > 0 {
		trans.log.Debugf("Generated new transforms: %v", trans)
	}

	return trans, nil
}

func (trans *Transforms) Add(t Transform) {
	if trans == nil {
		return
	}
	trans.List = append(trans.List, t)
}

// Run executes all transforms serially and returns the event and possibly
// an error.
func (trans *Transforms) Run(tr *Transformable) (*Transformable, error) {
	var err error
	for _, p := range trans.List {
		tr, err = p.Run(tr)
		if err != nil {
			return tr, errors.Wrapf(err, "failed applying transform %v", tr)
		}
	}
	return tr, nil
}

func (trans Transforms) String() string {
	var s []string
	for _, p := range trans.List {
		s = append(s, p.String())
	}
	return strings.Join(s, ", ")
}

func GetTargetInfo(t string) TargetInfo {
	parts := strings.SplitN(t, ".", 2)
	if len(parts) < 2 {
		return TargetInfo{}
	}
	switch parts[0] {
	case "url":
		if parts[1] == "value" {
			return TargetInfo{Type: TargetURLValue}
		}

		paramParts := strings.SplitN(parts[1], ".", 2)
		return TargetInfo{
			Type: TargetURLParams,
			Name: paramParts[1],
		}
	case "headers":
		return TargetInfo{
			Type: TargetHeaders,
			Name: parts[1],
		}
	case "body":
		return TargetInfo{
			Type: TargetBody,
			Name: parts[1],
		}
	case "cursor":
		return TargetInfo{
			Type: TargetCursor,
			Name: parts[1],
		}
	}
	return TargetInfo{}
}
