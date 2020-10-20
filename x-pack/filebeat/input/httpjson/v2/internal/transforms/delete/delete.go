package delete

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms"
	"github.com/pkg/errors"
)

const Name = "delete"

type config struct {
	Target string `config:"target"`
}

type delete struct {
	targetInfo transforms.TargetInfo

	run func(tr *transforms.Transformable, key string) error
}

func New(cfg *common.Config) (transforms.Transform, error) {
	c := &config{}
	if err := cfg.Unpack(c); err != nil {
		return nil, errors.Wrap(err, "fail to unpack the set configuration")
	}
	delete := &delete{
		targetInfo: transforms.GetTargetInfo(c.Target),
	}

	switch delete.targetInfo.Type {
	// case transforms.TargetCursor:
	case transforms.TargetBody:
		delete.run = runBody
	case transforms.TargetHeaders:
		delete.run = runHeader
	case transforms.TargetURLParams:
		delete.run = runURLParams
	case transforms.TargetURLValue:
		return nil, errors.New("can't append to url.value")
	default:
		return nil, errors.New("unknown target type")
	}

	return delete, nil
}

func (delete) String() string { return Name }

func (delete *delete) Run(tr *transforms.Transformable) (*transforms.Transformable, error) {
	return tr, delete.run(tr, delete.targetInfo.Name)
}

func deleteFromCommonMap(m common.MapStr, key string) error {
	if err := m.Delete(key); err != common.ErrKeyNotFound {
		return err
	}
	return nil
}

func runBody(tr *transforms.Transformable, key string) error {
	return deleteFromCommonMap(tr.Body, key)
}

func runHeader(tr *transforms.Transformable, key string) error {
	tr.Headers.Del(key)
	return nil
}

func runURLParams(tr *transforms.Transformable, key string) error {
	tr.URL.Query().Del(key)
	return nil
}
