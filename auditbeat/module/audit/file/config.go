package file

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
)

// HashType identifies a cryptographic algorithm.
type HashType string

// Unpack unpacks a string to a HashType for config parsing.
func (t *HashType) Unpack(v string) error {
	*t = HashType(v)
	return nil
}

var validHashes = []HashType{MD5, SHA1, SHA224, SHA256, SHA384, SHA3_224, SHA3_256, SHA3_384, SHA3_512, SHA512, SHA512_224, SHA512_256}

// Enum of hash types.
const (
	MD5        HashType = "md5"
	SHA1       HashType = "sha1"
	SHA224     HashType = "sha224"
	SHA256     HashType = "sha256"
	SHA384     HashType = "sha384"
	SHA3_224   HashType = "sha3_224"
	SHA3_256   HashType = "sha3_256"
	SHA3_384   HashType = "sha3_384"
	SHA3_512   HashType = "sha3_512"
	SHA512     HashType = "sha512"
	SHA512_224 HashType = "sha512_224"
	SHA512_256 HashType = "sha512_256"
)

// Config contains the configuration parameters for the file integrity
// metricset.
type Config struct {
	Paths               []string   `config:"file.paths" validate:"required"`
	HashTypes           []HashType `config:"file.hash_types"`
	MaxFileSize         string     `config:"file.max_file_size"`
	MaxFileSizeBytes    uint64     `config:",ignore"`
	ScanAtStart         bool       `config:"file.scan_at_start"`
	ScanRatePerSec      string     `config:"file.scan_rate_per_sec"`
	ScanRateBytesPerSec uint64     `config:",ignore"`

	// Recursive enables recursive monitoring of directories.
	// XXX: This feature is only implemented in the scanner. It needs to be
	// implemented in the fsnotify code. Don't use it yet.
	Recursive bool `config:"file.recursive"`
}

// Validate validates the config data and return an error explaining all the
// problems with the config. This method modifies the given config.
func (c *Config) Validate() error {
	// Resolve symlinks.
	for i, p := range c.Paths {
		if evalPath, err := filepath.EvalSymlinks(p); err == nil {
			c.Paths[i] = evalPath
		}
	}
	// Sort and deduplicate.
	sort.Strings(c.Paths)
	c.Paths = deduplicate(c.Paths)

	var errs multierror.Errors
	var err error

nextHash:
	for _, ht := range c.HashTypes {
		ht = HashType(strings.ToLower(string(ht)))
		for _, validHash := range validHashes {
			if ht == validHash {
				continue nextHash
			}
		}
		errs = append(errs, errors.Errorf("invalid file.hash_types value '%v'", ht))
	}

	c.MaxFileSizeBytes, err = humanize.ParseBytes(c.MaxFileSize)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "invalid file.max_file_size value"))
	} else if c.MaxFileSizeBytes <= 0 {
		errs = append(errs, errors.Errorf("file.max_file_size value (%v) must be positive", c.MaxFileSize))
	}

	c.ScanRateBytesPerSec, err = humanize.ParseBytes(c.ScanRatePerSec)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "invalid file.scan_rate_per_sec value"))
	}

	return errs.Err()
}

// deduplicate deduplicates the given sorted string slice. The returned slice
// reuses the same backing array as in (so don't use in after calling this).
func deduplicate(in []string) []string {
	var lastValue string
	out := in[:0]
	for _, value := range in {
		if value == lastValue {
			continue
		}
		out = append(out, value)
		lastValue = value
	}
	return out
}

var defaultConfig = Config{
	HashTypes:        []HashType{SHA1},
	MaxFileSize:      "100 MiB",
	MaxFileSizeBytes: 100 * 1024 * 1024,
	ScanAtStart:      true,
	ScanRatePerSec:   "50 MiB",
}
