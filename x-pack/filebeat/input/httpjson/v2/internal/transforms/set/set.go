package set

import (
	"net/url"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms"
	"github.com/pkg/errors"
)

const Name = "set"

type config struct {
	Target  string               `config:"target"`
	Value   *transforms.Template `config:"value"`
	Default string               `config:"default"`
}

type set struct {
	targetInfo   transforms.TargetInfo
	value        *transforms.Template
	defaultValue string

	run func(tr *transforms.Transformable, key, val string) error
}

func New(cfg *common.Config) (transforms.Transform, error) {
	c := &config{}
	if err := cfg.Unpack(c); err != nil {
		return nil, errors.Wrap(err, "fail to unpack the set configuration")
	}
	set := &set{
		targetInfo:   transforms.GetTargetInfo(c.Target),
		value:        c.Value,
		defaultValue: c.Default,
	}

	switch set.targetInfo.Type {
	// case transforms.TargetCursor:
	case transforms.TargetBody:
		set.run = runBody
	case transforms.TargetHeaders:
		set.run = runHeader
	case transforms.TargetURLValue:
		set.run = runURLValue
	case transforms.TargetURLParams:
		set.run = runURLParams
	default:
		return nil, errors.New("unknown target type")
	}

	return set, nil
}

func (set) String() string { return Name }

func (set *set) Run(tr *transforms.Transformable) (*transforms.Transformable, error) {
	value := set.value.Execute(tr, set.defaultValue)
	return tr, set.run(tr, set.targetInfo.Name, value)
}

func setToCommonMap(m common.MapStr, key, val string) error {
	if _, err := m.Put(key, val); err != nil {
		return err
	}
	return nil
}

func runBody(tr *transforms.Transformable, key, value string) error {
	return setToCommonMap(tr.Body, key, value)
}

func runHeader(tr *transforms.Transformable, key, value string) error {
	tr.Headers.Add(key, value)
	return nil
}

func runURLParams(tr *transforms.Transformable, key, value string) error {
	tr.URL.Query().Add(key, value)
	return nil
}

func runURLValue(tr *transforms.Transformable, _, value string) error {
	query := tr.URL.Query().Encode()
	url, err := url.Parse(value)
	if err != nil {
		return err
	}
	url.RawQuery = query
	tr.URL = *url
	return nil
}
