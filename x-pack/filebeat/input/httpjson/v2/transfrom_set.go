package v2

import (
	"fmt"
	httpURL "net/url"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/pkg/errors"
)

const setName = "set"

var (
	_ requestTransform    = &setRequest{}
	_ responseTransform   = &setResponse{}
	_ paginationTransform = &setPagination{}
)

type setConfig struct {
	Target  string    `config:"target"`
	Value   *valueTpl `config:"value"`
	Default string    `config:"default"`
}

type set struct {
	targetInfo   targetInfo
	value        *valueTpl
	defaultValue string

	run func(ctx transformContext, transformable *transformable, key, val string) error
}

func (set) transformName() string { return setName }

type setRequest struct {
	set
}

type setResponse struct {
	set
}

type setPagination struct {
	set
}

func newSetRequest(cfg *common.Config) (transform, error) {
	set, err := newSet(cfg)
	if err != nil {
		return nil, err
	}

	switch set.targetInfo.Type {
	case targetBody:
		set.run = setBody
	case targetHeader:
		set.run = setHeader
	case targetURLParams:
		set.run = setURLParams
	default:
		return nil, fmt.Errorf("invalid target type: %s", set.targetInfo.Type)
	}

	return &setRequest{set: set}, nil
}

func (setReq *setRequest) run(ctx transformContext, req *request) (*request, error) {
	transformable := &transformable{
		body:   req.body,
		header: req.header,
		url:    req.url,
	}
	if err := setReq.set.runSet(ctx, transformable); err != nil {
		return nil, err
	}
	return req, nil
}

func newSetResponse(cfg *common.Config) (transform, error) {
	set, err := newSet(cfg)
	if err != nil {
		return nil, err
	}

	switch set.targetInfo.Type {
	case targetBody:
		set.run = setBody
	default:
		return nil, fmt.Errorf("invalid target type: %s", set.targetInfo.Type)
	}

	return &setResponse{set: set}, nil
}

func (setRes *setResponse) run(ctx transformContext, res *response) (*response, error) {
	transformable := &transformable{
		body:   res.body,
		header: res.header,
		url:    res.url,
	}
	if err := setRes.set.runSet(ctx, transformable); err != nil {
		return nil, err
	}
	return res, nil
}

func newSetPagination(cfg *common.Config) (transform, error) {
	set, err := newSet(cfg)
	if err != nil {
		return nil, err
	}

	switch set.targetInfo.Type {
	case targetBody:
		set.run = setBody
	case targetHeader:
		set.run = setHeader
	case targetURLParams:
		set.run = setURLParams
	case targetURLValue:
		set.run = setURLValue
	default:
		return nil, fmt.Errorf("invalid target type: %s", set.targetInfo.Type)
	}

	return &setPagination{set: set}, nil
}

func (setPag *setPagination) run(ctx transformContext, pag *pagination) (*pagination, error) {
	transformable := &transformable{
		body:   pag.body,
		header: pag.header,
		url:    pag.url,
	}
	if err := setPag.set.runSet(ctx, transformable); err != nil {
		return nil, err
	}
	return pag, nil
}

func newSet(cfg *common.Config) (set, error) {
	c := &setConfig{}
	if err := cfg.Unpack(c); err != nil {
		return set{}, errors.Wrap(err, "fail to unpack the set configuration")
	}

	ti, err := getTargetInfo(c.Target)
	if err != nil {
		return set{}, err
	}

	return set{
		targetInfo:   ti,
		value:        c.Value,
		defaultValue: c.Default,
	}, nil
}

func (set *set) runSet(ctx transformContext, transformable *transformable) error {
	value := set.value.Execute(ctx, transformable, set.defaultValue)
	return set.run(ctx, transformable, set.targetInfo.Name, value)
}

func setToCommonMap(m common.MapStr, key, val string) error {
	if _, err := m.Put(key, val); err != nil {
		return err
	}
	return nil
}

func setBody(ctx transformContext, transformable *transformable, key, value string) error {
	return setToCommonMap(transformable.body, key, value)
}

func setHeader(ctx transformContext, transformable *transformable, key, value string) error {
	transformable.header.Add(key, value)
	return nil
}

func setURLParams(ctx transformContext, transformable *transformable, key, value string) error {
	q := transformable.url.Query()
	q.Add(key, value)
	transformable.url.RawQuery = q.Encode()
	return nil
}

func setURLValue(ctx transformContext, transformable *transformable, _, value string) error {
	query := transformable.url.Query().Encode()
	url, err := httpURL.Parse(value)
	if err != nil {
		return err
	}
	url.RawQuery = query
	transformable.url = url
	return nil
}
