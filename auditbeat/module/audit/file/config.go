package file

import (
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
)

type Config struct {
	Paths            []string `config:"file.paths" validate:"required"`
	HashTypes        []string `config:"file.hash_types"`
	MaxFileSize      string   `config:"file.max_file_size"`
	MaxFileSizeBytes uint64   `config:",ignore"`
}

func (c *Config) Validate() error {
	var errs multierror.Errors
	var err error

	c.MaxFileSizeBytes, err = humanize.ParseBytes(c.MaxFileSize)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "invalid file.max_file_size value"))
	}

	for _, ht := range c.HashTypes {
		switch strings.ToLower(ht) {
		case "md5", "sha1", "sha224", "sha256", "sha384", "sha512", "sha512_224", "sha512_256":
		default:
			errs = append(errs, errors.Errorf("invalid hash type '%v'", ht))
		}
	}

	return errs.Err()
}

var defaultConfig = Config{
	MaxFileSize:      "100 MiB",
	MaxFileSizeBytes: 100 * 1024 * 1024,
	HashTypes:        []string{"sha1"},
}
