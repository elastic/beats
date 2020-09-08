// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	downloader "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/localremote"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/hashicorp/go-multierror"
)

const (
	agentName         = "elastic-agent"
	hashLen           = 6
	agentCommitFile   = ".elastic-agent.active.commit"
	agentArtifactName = "beats/" + agentName
)

// After running Upgrade agent should download its own version specified by action
// from repository specified by fleet.
type handlerUpgrade struct {
	settings *artifact.Config
	log      *logger.Logger
	closers  []context.CancelFunc
}

func (h *handlerUpgrade) Handle(ctx context.Context, a action, acker fleetAcker) error {
	h.log.Debugf("handlerUpgrade: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionUpgrade)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionUpgrade and received %T", a)
	}

	archivePath, err := h.downloadArtifact(ctx, action)
	if err != nil {
		return err
	}

	newHash, err := h.unpack(ctx, action, archivePath)
	if err != nil {
		return err
	}

	if newHash == "" {
		return errors.New("unknown hash")
	}

	if err := h.changeSymlink(ctx, action, newHash); err != nil {
		return err
	}

	if err := h.markUpgrade(ctx, action); err != nil {
		return err
	}

	return h.reexec(ctx, action)
}

func (h *handlerUpgrade) downloadArtifact(ctx context.Context, action *fleetapi.ActionUpgrade) (string, error) {
	// do not update source config
	settings := *h.settings
	if action.SourceURI != "" {
		settings.SourceURI = action.SourceURI
	}

	fetcher := downloader.NewDownloader(h.log, &settings)
	verifier, err := downloader.NewVerifier(h.log, &settings)
	if err != nil {
		return "", errors.New(err, "initiating verifier")
	}

	path, err := fetcher.Download(ctx, agentName, agentArtifactName, action.Version)
	if err != nil {
		return "", errors.New(err, "failed upgrade of agent binary")
	}

	matches, err := verifier.Verify(agentName, action.Version)
	if err != nil {
		return "", errors.New(err, "failed verification of agent binary")
	}
	if !matches {
		return "", errors.New("failed verification of agent binary, hash does not match", errors.TypeSecurity)
	}

	return path, nil
}

// untar unpacks archive correctly, skips root (symlink, config...) unpacks data/*
func (h *handlerUpgrade) unpack(ctx context.Context, action *fleetapi.ActionUpgrade, archivePath string) (string, error) {
	// unpack must occur in directory that holds the installation directory
	// or the extraction will be double nested
	var hash string
	var err error
	if runtime.GOOS == "windows" {
		hash, err = unzip(action, archivePath)
	} else {
		hash, err = untar(action, archivePath)
	}
	if err != nil {
		return "", err
	}

	return hash, nil
}

// changeSymlink changes root symlink so it points to updated version
func (h *handlerUpgrade) changeSymlink(ctx context.Context, action *fleetapi.ActionUpgrade, newHash string) error {
	return errors.New("not yet implemented")
}

// markUpgrade marks update happened so we can handle grace period
func (h *handlerUpgrade) markUpgrade(ctx context.Context, action *fleetapi.ActionUpgrade) error {
	return errors.New("not yet implemented")
}

// reexec restarts agent so new version is run
func (h *handlerUpgrade) reexec(ctx context.Context, action *fleetapi.ActionUpgrade) error {
	return errors.New("not yet implemented")
}

func unzip(action *fleetapi.ActionUpgrade, archivePath string) (string, error) {
	var hash, rootDir string
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	unpackFile := func(f *zip.File) (err error) {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if cerr := rc.Close(); cerr != nil {
				err = multierror.Append(err, cerr)
			}
		}()

		//get hash
		if f.Name == agentCommitFile {
			hashBytes, err := ioutil.ReadAll(rc)
			if err != nil || len(hashBytes) < hashLen {
				return err
			}

			hash = string(hashBytes[:hashLen])
			return nil
		}

		// skip everything outside data/
		if !strings.HasPrefix(f.Name, "data/") {
			return nil
		}

		path := filepath.Join(paths.Data(), strings.TrimPrefix(f.Name, "data/"))

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if cerr := f.Close(); cerr != nil {
					err = multierror.Append(err, cerr)
				}
			}()

			if _, err = io.Copy(f, rc); err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		// TODO: verify if needed
		if rootDir == "" && filepath.Base(f.Name) == filepath.Dir(f.Name) {
			return f.Name, nil
		}
		if currentDir := filepath.Dir(f.Name); rootDir == "" || len(currentDir) < len(rootDir) {
			rootDir = currentDir
		}
		// EOT

		if err := unpackFile(f); err != nil {
			return "", err
		}
	}

	// if root directory is not the same as desired directory rename
	// e.g contains `-windows-` or  `-SNAPSHOT-`
	if dataPath := paths.Data(); rootDir != dataPath {
		if err := os.Rename(rootDir, dataPath); err != nil {
			return "", errors.New(err, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, installDir))
		}
	}

	return hash, nil

}

func untar(action *fleetapi.ActionUpgrade, archivePath string) (string, error) {
	r, err := os.Open(archivePath)
	if err != nil {
		return "", errors.New(fmt.Sprintf("artifact for 'elastic-agent' version '%s' could not be found at '%s'", action.Version, archivePath), errors.TypeFilesystem, errors.M(errors.MetaKeyPath, archivePath))
	}
	defer r.Close()

	zr, err := gzip.NewReader(r)
	if err != nil {
		return "", errors.New("requires gzip-compressed body", err, errors.TypeFilesystem)
	}

	tr := tar.NewReader(zr)
	var rootDir string
	var hash string

	for {
		f, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if !validFileName(f.Name) {
			return "", errors.New("tar contained invalid filename: %q", f.Name, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, f.Name))
		}

		//get hash
		if f.Name == agentCommitFile {
			hashBytes, err := ioutil.ReadAll(tr)
			if err != nil || len(hashBytes) < hashLen {
				return "", err
			}

			hash = string(hashBytes[:hashLen])
			continue
		}

		// skip everything outside data/
		if !strings.HasPrefix(f.Name, "data/") {
			continue
		}

		rel := filepath.FromSlash(strings.TrimPrefix(f.Name, "data/"))
		abs := filepath.Join(paths.Data(), rel)

		// find the root dir
		if currentDir := filepath.Dir(abs); rootDir == "" || len(filepath.Dir(rootDir)) > len(currentDir) {
			rootDir = currentDir
		}

		fi := f.FileInfo()
		mode := fi.Mode()
		switch {
		case mode.IsRegular():
			// just to be sure, it should already be created by Dir type
			if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
				return "", errors.New(err, "TarInstaller: creating directory for file "+abs, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, abs))
			}

			wf, err := os.OpenFile(abs, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode.Perm())
			if err != nil {
				return "", errors.New(err, "TarInstaller: creating file "+abs, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, abs))
			}

			_, err = io.Copy(wf, tr)
			if closeErr := wf.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				return "", fmt.Errorf("TarInstaller: error writing to %s: %v", abs, err)
			}
		case mode.IsDir():
			if err := os.MkdirAll(abs, 0755); err != nil {
				return "", errors.New(err, "TarInstaller: creating directory for file "+abs, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, abs))
			}
		default:
			return "", errors.New(fmt.Sprintf("tar file entry %s contained unsupported file type %v", f.Name, mode), errors.TypeFilesystem, errors.M(errors.MetaKeyPath, f.Name))
		}
	}

	return hash, nil
}

func validFileName(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}

func isSubdir(base, target string) (bool, error) {
	relPath, err := filepath.Rel(base, target)
	return strings.HasPrefix(relPath, ".."), err
}

func installDir(ctx context.Context, action *fleetapi.ActionUpgrade, cfg *artifact.Config) string {
	agentSpec := program.Spec{}
	d := app.NewDescriptor(agentSpec, action.Version, cfg, map[app.Tag]string{})

	return d.Directory()
}
