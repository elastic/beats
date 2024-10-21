// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/elastic-agent/testing/upgradetest"
)

func TestStandaloneUpgradeRetryDownload(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Upgrade,
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
	})

	ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
	defer cancel()

	// Start at the build version as we want to test the retry
	// logic that is in the build.
	startFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	// The end version does not matter much but it must not match
	// the commit hash of the current build.
	endVersion, err := upgradetest.PreviousMinor()
	require.NoError(t, err)
	endFixture, err := atesting.NewFixture(
		t,
		endVersion.String(),
		atesting.WithFetcher(atesting.ArtifactFetcher()),
	)
	require.NoError(t, err)

	// uses an internal http server that returns bad requests
	// until it returns a successful request
	srcPackage, err := endFixture.SrcPackage(ctx)
	require.NoError(t, err)

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port

	count := 0
	fs := http.FileServer(http.Dir(filepath.Dir(srcPackage)))
	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		// fix path to remove '/beats/elastic-agent/' prefix
		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
		}
		if strings.HasPrefix(upath, "/beats/elastic-agent/") {
			upath = strings.TrimPrefix(upath, "/beats/elastic-agent/")
		}
		r.URL.Path = upath

		if path.Base(r.URL.Path) == filepath.Base(srcPackage) && count < 2 {
			// first 2 requests return 404
			count += 1
			t.Logf("request #%d; returning not found", count)
			rw.WriteHeader(http.StatusNotFound)
			return
		}

		fs.ServeHTTP(rw, r)
	})

	go func() {
		_ = http.Serve(l, handler)
	}()

	sourceURI := fmt.Sprintf("http://localhost:%d", port)
	err = upgradetest.PerformUpgrade(
		ctx, startFixture, endFixture, t, upgradetest.WithSourceURI(sourceURI))
	assert.NoError(t, err)
	assert.Equal(t, 2, count, "retry request didn't occur")
}
