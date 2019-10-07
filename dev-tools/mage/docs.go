// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mage

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"

	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
)

const (
	elasticDocsRepoURL = "https://github.com/elastic/docs.git"
)

type docsBuilder struct{}

type asciidocParams struct {
	name      string
	indexFile string
}

// DocsOption is a documentation generation option for controlling how the docs
// are built.
type DocsOption func(params *asciidocParams)

// DocsName specifies the documentation's name (default to BeatName).
func DocsName(name string) DocsOption {
	return func(params *asciidocParams) {
		params.name = name
	}
}

// DocsIndexFile specifies the index file (defaults to docs/index.asciidoc).
func DocsIndexFile(file string) DocsOption {
	return func(params *asciidocParams) {
		params.indexFile = file
	}
}

// Docs holds the utilities for building documentation.
var Docs = docsBuilder{}

// FieldDocs generates docs/fields.asciidoc from the specified fields.yml file.
func (docsBuilder) FieldDocs(fieldsYML string) error {
	// Run the docs_collector.py script.
	ve, err := PythonVirtualenv()
	if err != nil {
		return err
	}

	python, err := LookVirtualenvPath(ve, "python")
	if err != nil {
		return err
	}

	esBeats, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	// TODO: Port this script to Go.
	log.Println(">> Generating docs/fields.asciidoc for", BeatName)
	return sh.Run(python, LibbeatDir("scripts/generate_fields_docs.py"),
		fieldsYML,                     // Path to fields.yml.
		BeatName,                      // Beat title.
		esBeats,                       // Path to general beats folder.
		"--output_path", OSSBeatDir()) // It writes to {output_path}/docs/fields.asciidoc.
}

func (b docsBuilder) AsciidocBook(opts ...DocsOption) error {
	params := asciidocParams{
		name:      BeatName,
		indexFile: CWD("docs/index.asciidoc"),
	}
	for _, opt := range opts {
		opt(&params)
	}

	repo, err := GetProjectRepoInfo()
	if err != nil {
		return err
	}

	cloneDir := CreateDir(filepath.Join(repo.RootDir, "build/elastic_docs_repo"))

	// Clone if elastic_docs_repo does not exist.
	if _, err := os.Stat(cloneDir); err != nil {
		log.Println("Cloning elastic/docs to", cloneDir)
		if err = sh.Run("git", "clone", "--depth=1", elasticDocsRepoURL, cloneDir); err != nil {
			return err
		}
	} else {
		log.Println("Using existing elastic/docs at", cloneDir)
	}

	// Render HTML.
	htmlDir := CWD("build/html_docs", params.name)
	buildDocsScript := filepath.Join(cloneDir, "build_docs")
	args := []string{
		"--chunk=1",
		"--doc", params.indexFile,
		"--out", htmlDir,
	}
	fmt.Println(">> Building HTML docs at", filepath.Join(htmlDir, "index.html"))
	if err := sh.Run(buildDocsScript, args...); err != nil {
		return err
	}

	// Serve docs with and HTTP server and open the browser.
	if preview, _ := strconv.ParseBool(os.Getenv("PREVIEW")); preview {
		srv := b.servePreview(htmlDir)
		url := "http://" + srv.Addr
		fmt.Println("Serving docs preview at", url)
		b.openBrowser(url)

		// Wait
		fmt.Println("Ctrl+C to stop")
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		srv.Shutdown(context.Background())
	}
	return nil
}

// open opens the specified URL in the default browser.
func (docsBuilder) openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func (docsBuilder) servePreview(dir string) *http.Server {
	server := &http.Server{
		Addr:    net.JoinHostPort("localhost", EnvOr("PREVIEW_PORT", "8000")),
		Handler: http.FileServer(http.Dir(dir)),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(errors.Wrap(err, "failed to start docs preview"))
		}
	}()

	return server
}
