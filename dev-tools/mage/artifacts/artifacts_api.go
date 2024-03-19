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

package artifacts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"

	"github.com/elastic/beats/v7/dev-tools/mage/version"
)

const (
	defaultArtifactAPIURL = "https://artifacts-api.elastic.co/"

	artifactsAPIV1VersionsEndpoint      = "v1/versions/"
	artifactsAPIV1VersionBuildsEndpoint = "v1/versions/%s/builds/"
	artifactAPIV1BuildDetailsEndpoint   = "v1/versions/%s/builds/%s"
	// artifactAPIV1SearchVersionPackage = "v1/search/%s/%s"
)

var (
	ErrLatestVersionNil        = errors.New("latest version is nil")
	ErrSnapshotVersionsEmpty   = errors.New("snapshot list is nil")
	ErrInvalidVersionRetrieved = errors.New("invalid version retrieved from artifact API")

	ErrBadHTTPStatusCode = errors.New("bad http status code")
)

type Manifests struct {
	LastUpdateTime         string `json:"last-update-time"`
	SecondsSinceLastUpdate int    `json:"seconds-since-last-update"`
}

type VersionList struct {
	Versions  []string  `json:"versions"`
	Aliases   []string  `json:"aliases"`
	Manifests Manifests `json:"manifests"`
}

type VersionBuilds struct {
	Builds    []string  `json:"builds"`
	Manifests Manifests `json:"manifests"`
}

type Package struct {
	URL          string   `json:"url"`
	ShaURL       string   `json:"sha_url"`
	AscURL       string   `json:"asc_url"`
	Type         string   `json:"type"`
	Architecture string   `json:"architecture"`
	Os           []string `json:"os"`
	Classifier   string   `json:"classifier"`
	Attributes   struct {
		IncludeInRepo string `json:"include_in_repo"`
		ArtifactNoKpi string `json:"artifactNoKpi"`
		Internal      string `json:"internal"`
		ArtifactID    string `json:"artifact_id"`
		Oss           string `json:"oss"`
		Group         string `json:"group"`
	} `json:"attributes"`
}

type Dependency struct {
	Prefix   string `json:"prefix"`
	BuildUri string `json:"build_uri"`
}

type Project struct {
	Branch                       string             `json:"branch"`
	CommitHash                   string             `json:"commit_hash"`
	CommitURL                    string             `json:"commit_url"`
	ExternalArtifactsManifestURL string             `json:"external_artifacts_manifest_url"`
	BuildDurationSeconds         int                `json:"build_duration_seconds"`
	Packages                     map[string]Package `json:"packages"`
	Dependencies                 []Dependency       `json:"dependencies"`
}

type Build struct {
	Projects             map[string]Project `json:"projects"`
	StartTime            string             `json:"start_time"`
	ReleaseBranch        string             `json:"release_branch"`
	Prefix               string             `json:"prefix"`
	EndTime              string             `json:"end_time"`
	ManifestVersion      string             `json:"manifest_version"`
	Version              string             `json:"version"`
	Branch               string             `json:"branch"`
	BuildID              string             `json:"build_id"`
	BuildDurationSeconds int                `json:"build_duration_seconds"`
}

type BuildDetails struct {
	Build     Build
	Manifests Manifests `json:"manifests"`
}

type SearchPackageResult struct {
	Packages  map[string]Package `json:"packages"`
	Manifests Manifests          `json:"manifests"`
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type ArtifactAPIClientOpt func(aac *ArtifactAPIClient)

func WithUrl(url string) ArtifactAPIClientOpt {
	return func(aac *ArtifactAPIClient) { aac.url = url }
}

func WithHttpClient(client httpDoer) ArtifactAPIClientOpt {
	return func(aac *ArtifactAPIClient) { aac.c = client }
}

// ArtifactAPIClient is a small (and incomplete) client for the Elastic artifact API.
// More information about the API can be found at https://artifacts-api.elastic.co/v1
// which will print a list of available operations
type ArtifactAPIClient struct {
	c   httpDoer
	url string
}

// NewArtifactAPIClient creates a new Artifact API client
func NewArtifactAPIClient(opts ...ArtifactAPIClientOpt) *ArtifactAPIClient {
	c := &ArtifactAPIClient{
		url: defaultArtifactAPIURL,
		c:   new(http.Client),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// GetVersions returns a list of versions as server by the Artifact API along with some aliases and manifest information
func (aac ArtifactAPIClient) GetVersions(ctx context.Context) (list *VersionList, err error) {
	joinedURL, err := aac.composeURL(artifactsAPIV1VersionsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("couldn't compose URL: %w", err)
	}

	resp, err := aac.createAndPerformRequest(ctx, joinedURL)
	if err != nil {
		return nil, fmt.Errorf("getting versions: %w", err)
	}
	defer resp.Body.Close()

	return checkResponseAndUnmarshal[VersionList](resp)
}

// GetBuildsForVersion returns a list of builds for a specific version.
// version should be one of the version strings returned by the GetVersions (expected format is semver
// with optional prerelease but no build metadata, for example 8.9.0-SNAPSHOT)
func (aac ArtifactAPIClient) GetBuildsForVersion(ctx context.Context, version string) (builds *VersionBuilds, err error) {
	joinedURL, err := aac.composeURL(fmt.Sprintf(artifactsAPIV1VersionBuildsEndpoint, version))
	if err != nil {
		return nil, fmt.Errorf("couldn't compose URL: %w", err)
	}

	resp, err := aac.createAndPerformRequest(ctx, joinedURL)
	if err != nil {
		return nil, fmt.Errorf("getting builds for version %s: %w", version, err)
	}
	defer resp.Body.Close()

	return checkResponseAndUnmarshal[VersionBuilds](resp)
}

// GetBuildDetails returns the list of project and artifacts related to a specific build.
// Version parameter format follows semver (without build metadata) and buildID format is <major>.<minor>.<patch>-<buildhash> as returned by
// GetBuildsForVersion()
func (aac ArtifactAPIClient) GetBuildDetails(ctx context.Context, version string, buildID string) (buildDetails *BuildDetails, err error) {
	joinedURL, err := aac.composeURL(fmt.Sprintf(artifactAPIV1BuildDetailsEndpoint, version, buildID))
	if err != nil {
		return nil, fmt.Errorf("couldn't compose URL: %w", err)
	}

	resp, err := aac.createAndPerformRequest(ctx, joinedURL)
	if err != nil {
		return nil, fmt.Errorf("getting build details for version %s buildID %s: %w", version, buildID, err)
	}
	defer resp.Body.Close()

	return checkResponseAndUnmarshal[BuildDetails](resp)
}

func (aac ArtifactAPIClient) composeURL(relativePath string) (string, error) {
	joinedURL, err := url.JoinPath(aac.url, relativePath)
	if err != nil {
		return "", fmt.Errorf("composing URL with %q %q: %w", aac.url, relativePath, err)
	}

	return joinedURL, nil
}

func (aac ArtifactAPIClient) createAndPerformRequest(ctx context.Context, URL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, URL, nil)
	if err != nil {
		err = fmt.Errorf("composing request: %w", err)
		return nil, err
	}

	resp, err := aac.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing http request %v: %w", req, err)
	}

	return resp, nil
}

func checkResponseAndUnmarshal[T any](resp *http.Response) (*T, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%d: %w", resp.StatusCode, ErrBadHTTPStatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	result := new(T)
	err = json.Unmarshal(respBytes, result)

	if err != nil {
		return nil, fmt.Errorf("unmarshaling: %w", err)
	}

	return result, nil
}

type logger interface {
	Logf(format string, args ...any)
}

func (aac ArtifactAPIClient) GetLatestSnapshotVersion(ctx context.Context, log logger) (*version.ParsedSemVer, error) {
	vList, err := aac.GetVersions(ctx)
	if err != nil {
		return nil, err
	}

	if vList == nil {
		return nil, ErrSnapshotVersionsEmpty
	}

	sortedParsedVersions := make(version.SortableParsedVersions, 0, len(vList.Versions))
	for _, v := range vList.Versions {
		pv, err := version.ParseVersion(v)
		if err != nil {
			log.Logf("invalid version retrieved from artifact API: %q", v)
			return nil, ErrInvalidVersionRetrieved
		}
		sortedParsedVersions = append(sortedParsedVersions, pv)
	}

	if len(sortedParsedVersions) == 0 {
		return nil, ErrSnapshotVersionsEmpty
	}

	// normally the output of the versions returned by artifact API is already
	// sorted in ascending order.If we want to sort in descending order we need
	// to pass a sort.Reverse to sort.Sort.
	sort.Sort(sort.Reverse(sortedParsedVersions))

	var latestSnapshotVersion *version.ParsedSemVer
	// fetch the latest SNAPSHOT build
	for _, pv := range sortedParsedVersions {
		if pv.IsSnapshot() {
			latestSnapshotVersion = pv
			break
		}
	}
	if latestSnapshotVersion == nil {
		return nil, ErrLatestVersionNil
	}
	return latestSnapshotVersion, nil
}
