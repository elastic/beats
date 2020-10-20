package append

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms"
	"github.com/pkg/errors"
)

const Name = "append"

type config struct {
	Target  string               `config:"target"`
	Value   *transforms.Template `config:"value"`
	Default string               `config:"default"`
}

type appendTransform struct {
	targetInfo   transforms.TargetInfo
	value        *transforms.Template
	defaultValue string

	run func(tr *transforms.Transformable, key, val string) error
}

func New(cfg *common.Config) (transforms.Transform, error) {
	c := &config{}
	if err := cfg.Unpack(c); err != nil {
		return nil, errors.Wrap(err, "fail to unpack the append configuration")
	}
	app := &appendTransform{
		targetInfo:   transforms.GetTargetInfo(c.Target),
		value:        c.Value,
		defaultValue: c.Default,
	}

	switch app.targetInfo.Type {
	// case transforms.TargetCursor:
	case transforms.TargetBody:
		app.run = runBody
	case transforms.TargetHeaders:
		app.run = runHeader
	case transforms.TargetURLParams:
		app.run = runURLParams
	case transforms.TargetURLValue:
		return nil, errors.New("can't append to url.value")
	default:
		return nil, errors.New("unknown target type")
	}

	return app, nil
}

func (appendTransform) String() string { return Name }

func (app *appendTransform) Run(tr *transforms.Transformable) (*transforms.Transformable, error) {
	value := app.value.Execute(tr, app.defaultValue)
	return tr, app.run(tr, app.targetInfo.Name, value)
}

func appendToCommonMap(m common.MapStr, key, val string) error {
	var value interface{} = val
	if found, _ := m.HasKey(key); found {
		prev, _ := m.GetValue(key)
		switch t := prev.(type) {
		case []string:
			value = append(t, val)
		case []interface{}:
			value = append(t, val)
		default:
			value = []interface{}{prev, val}
		}

	}
	if _, err := m.Put(key, value); err != nil {
		return err
	}
	return nil
}

func runBody(tr *transforms.Transformable, key, value string) error {
	return appendToCommonMap(tr.Body, key, value)
}

func runHeader(tr *transforms.Transformable, key, value string) error {
	tr.Headers.Add(key, value)
	return nil
}

func runURLParams(tr *transforms.Transformable, key, value string) error {
	tr.URL.Query().Add(key, value)
	return nil
}
