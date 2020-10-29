package v2

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/pkg/errors"
)

const deleteName = "delete"

var (
	_ requestTransform    = &deleteRequest{}
	_ responseTransform   = &deleteResponse{}
	_ paginationTransform = &deletePagination{}
)

type deleteConfig struct {
	Target string `config:"target"`
}

type delete struct {
	targetInfo targetInfo

	run func(ctx transformContext, transformable *transformable, key string) error
}

func (delete) transformName() string { return deleteName }

type deleteRequest struct {
	delete
}

type deleteResponse struct {
	delete
}

type deletePagination struct {
	delete
}

func newDeleteRequest(cfg *common.Config) (transform, error) {
	delete, err := newDelete(cfg)
	if err != nil {
		return nil, err
	}

	switch delete.targetInfo.Type {
	case targetBody:
		delete.run = deleteBody
	case targetHeader:
		delete.run = deleteHeader
	case targetURLParams:
		delete.run = deleteURLParams
	default:
		return nil, fmt.Errorf("invalid target type: %s", delete.targetInfo.Type)
	}

	return &deleteRequest{delete: delete}, nil
}

func (deleteReq *deleteRequest) run(ctx transformContext, req *request) (*request, error) {
	transformable := &transformable{
		body:   req.body,
		header: req.header,
		url:    req.url,
	}
	if err := deleteReq.delete.runDelete(ctx, transformable); err != nil {
		return nil, err
	}
	return req, nil
}

func newDeleteResponse(cfg *common.Config) (transform, error) {
	delete, err := newDelete(cfg)
	if err != nil {
		return nil, err
	}

	switch delete.targetInfo.Type {
	case targetBody:
		delete.run = deleteBody
	default:
		return nil, fmt.Errorf("invalid target type: %s", delete.targetInfo.Type)
	}

	return &deleteResponse{delete: delete}, nil
}

func (deleteRes *deleteResponse) run(ctx transformContext, res *response) (*response, error) {
	transformable := &transformable{
		body:   res.body,
		header: res.header,
		url:    res.url,
	}
	if err := deleteRes.delete.runDelete(ctx, transformable); err != nil {
		return nil, err
	}
	return res, nil
}

func newDeletePagination(cfg *common.Config) (transform, error) {
	delete, err := newDelete(cfg)
	if err != nil {
		return nil, err
	}

	switch delete.targetInfo.Type {
	case targetBody:
		delete.run = deleteBody
	case targetHeader:
		delete.run = deleteHeader
	case targetURLParams:
		delete.run = deleteURLParams
	default:
		return nil, fmt.Errorf("invalid target type: %s", delete.targetInfo.Type)
	}

	return &deletePagination{delete: delete}, nil
}

func (deletePag *deletePagination) run(ctx transformContext, pag *pagination) (*pagination, error) {
	transformable := &transformable{
		body:   pag.body,
		header: pag.header,
		url:    pag.url,
	}
	if err := deletePag.delete.runDelete(ctx, transformable); err != nil {
		return nil, err
	}
	return pag, nil
}

func newDelete(cfg *common.Config) (delete, error) {
	c := &deleteConfig{}
	if err := cfg.Unpack(c); err != nil {
		return delete{}, errors.Wrap(err, "fail to unpack the delete configuration")
	}

	ti, err := getTargetInfo(c.Target)
	if err != nil {
		return delete{}, err
	}

	return delete{
		targetInfo: ti,
	}, nil
}

func (delete *delete) runDelete(ctx transformContext, transformable *transformable) error {
	return delete.run(ctx, transformable, delete.targetInfo.Name)
}

func deleteFromCommonMap(m common.MapStr, key string) error {
	if err := m.Delete(key); err != common.ErrKeyNotFound {
		return err
	}
	return nil
}

func deleteBody(ctx transformContext, transformable *transformable, key string) error {
	return deleteFromCommonMap(transformable.body, key)
}

func deleteHeader(ctx transformContext, transformable *transformable, key string) error {
	transformable.header.Del(key)
	return nil
}

func deleteURLParams(ctx transformContext, transformable *transformable, key string) error {
	transformable.url.Query().Del(key)
	return nil
}
