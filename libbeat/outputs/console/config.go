package console

import "github.com/elastic/beats/libbeat/common/fmtstr"

type config struct {
	Pretty bool                      `config:"pretty"`
	Format *fmtstr.EventFormatString `config:"format"`
}

var (
	defaultConfig = config{
		Pretty: false,
		Format: nil,
	}
)
