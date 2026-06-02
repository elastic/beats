// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package artifactformat

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/msiutil"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/pkgutil"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/tar"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/zip"
)

type Format string

const (
	Unknown Format = ""
	TarGz   Format = "tar.gz"
	Zip     Format = "zip"
	Pkg     Format = "pkg"
	Msi     Format = "msi"
)

func Detect(pathOrURL string) (Format, error) {
	path := pathOrURL
	if parsedURL, err := url.Parse(pathOrURL); err == nil && parsedURL.Path != "" {
		path = parsedURL.Path
	}
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return TarGz, nil
	case strings.HasSuffix(lower, ".zip"):
		return Zip, nil
	case strings.HasSuffix(lower, ".pkg"):
		return Pkg, nil
	case strings.HasSuffix(lower, ".msi"):
		return Msi, nil
	default:
		return Unknown, fmt.Errorf("unsupported artifact format for %q", pathOrURL)
	}
}

func ExtractAll(format Format, artifactFile, destinationDir string) error {
	switch format {
	case TarGz:
		return tar.ExtractFile(artifactFile, destinationDir)
	case Zip:
		return zip.UnzipFile(artifactFile, destinationDir)
	case Pkg:
		return pkgutil.Expand(artifactFile, destinationDir)
	case Msi:
		return msiutil.Expand(artifactFile, destinationDir)
	default:
		return fmt.Errorf("unsupported artifact format %q", format)
	}
}

// ExtractAllSkipEscaping is like ExtractAll but silently skips symlink/hardlink
// entries whose targets escape the destination directory. This is used for
// runtime custom artifact extraction where official tarballs may contain
// absolute symlinks (e.g. usr/bin/osqueryd -> /opt/osquery/bin/osqueryd).
func ExtractAllSkipEscaping(format Format, artifactFile, destinationDir string) error {
	switch format {
	case TarGz:
		return tar.ExtractFileSkipEscaping(artifactFile, destinationDir)
	case Zip:
		return zip.UnzipFile(artifactFile, destinationDir)
	case Pkg:
		return pkgutil.Expand(artifactFile, destinationDir)
	case Msi:
		return msiutil.Expand(artifactFile, destinationDir)
	default:
		return fmt.Errorf("unsupported artifact format %q", format)
	}
}
