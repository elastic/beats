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

package kubernetes

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/common"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type DockerConfig struct {
	CurrentContext string `json:"currentContext"`
}

type DockerContext struct {
	Name      string                 `json:"Name"`
	Metadata  map[string]interface{} `json:"Metadata"`
	Endpoints map[string]Endpoint    `json:"Endpoints"`
	Storage   map[string]interface{} `json:"Storage"`
	TLS       bool                   `json:"TLS"`
}

type DockerBuildOutput struct {
	Stream string `json:"stream"`
	Aux    struct {
		ID string `json:"ID"`
	} `json:"aux"`
}

type Endpoint struct {
	Host string `json:"Host"`
}

// AddK8STestsToImage compiles and adds the k8s-inner-tests binary to the given image
func AddK8STestsToImage(ctx context.Context, logger common.Logger, baseImage string, arch string) (string, error) {
	// compile k8s test with tag kubernetes_inner
	buildBase, err := filepath.Abs("build")
	if err != nil {
		return "", err
	}

	testBinary := filepath.Join(buildBase, "k8s-inner-tests")

	params := devtools.GoTestArgs{
		TestName:   "k8s-inner-tests",
		Race:       false,
		Packages:   []string{"./testing/kubernetes_inner/..."},
		Tags:       []string{"kubernetes_inner"},
		OutputFile: testBinary,
		Env: map[string]string{
			"GOOS":        "linux",
			"GOARCH":      arch,
			"CGO_ENABLED": "0",
		},
	}

	if err := devtools.GoTestBuild(ctx, params); err != nil {
		return "", err
	}

	cli, err := getDockerClient()
	if err != nil {
		return "", err
	}

	// dockerfile to just copy the tests binary
	dockerfile := fmt.Sprintf(`
	FROM %s
	COPY testsBinary /usr/share/elastic-agent/k8s-inner-tests
	`, baseImage)

	// Create a tar archive with the Dockerfile and the binary
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add Dockerfile to tar
	err = tw.WriteHeader(&tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfile)),
	})
	if err != nil {
		return "", err
	}
	_, err = tw.Write([]byte(dockerfile))
	if err != nil {
		return "", err
	}

	// Add binary to tar
	binaryFile, err := os.Open(testBinary)
	if err != nil {
		return "", err
	}
	defer binaryFile.Close()

	info, err := binaryFile.Stat()
	if err != nil {
		return "", err
	}

	err = tw.WriteHeader(&tar.Header{
		Name: "testsBinary",
		Mode: 0777,
		Size: info.Size(),
	})
	if err != nil {
		return "", err
	}
	_, err = io.Copy(tw, binaryFile)
	if err != nil {
		return "", err
	}

	err = tw.Close()
	if err != nil {
		return "", err
	}

	outputImage := baseImage + "-tests"

	// Build the image
	imageBuildResponse, err := cli.ImageBuild(ctx, &buf, types.ImageBuildOptions{
		Tags:       []string{outputImage},
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		return "", err
	}
	defer imageBuildResponse.Body.Close()

	scanner := bufio.NewScanner(imageBuildResponse.Body)
	for scanner.Scan() {
		line := scanner.Text()
		var output DockerBuildOutput
		if err := json.Unmarshal([]byte(line), &output); err != nil {
			return "", fmt.Errorf("error at parsing JSON: %w", err)
		}

		if output.Stream != "" {
			if out := strings.TrimRight(output.Stream, "\n"); out != "" {
				logger.Logf(out)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return outputImage, nil
}

// getDockerClient returns an instance of the Docker client. It first checks
// if there is a current context inside $/.docker/config.json and instantiates
// a client based on it. Otherwise, it fallbacks to a docker client with values
// from environment variables.
func getDockerClient() (*client.Client, error) {

	envClient := func() (*client.Client, error) {
		return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	}

	type DockerConfig struct {
		CurrentContext string `json:"currentContext"`
	}

	configFile := filepath.Join(os.Getenv("HOME"), ".docker", "config.json")
	file, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return envClient()
		}
		return nil, err
	}
	defer file.Close()

	var config DockerConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	if config.CurrentContext == "" {
		return envClient()
	}

	contextDir := filepath.Join(os.Getenv("HOME"), ".docker", "contexts", "meta")
	files, err := os.ReadDir(contextDir)
	if err != nil {
		if os.IsNotExist(err) {
			return envClient()
		}
		return nil, fmt.Errorf("unable to read Docker contexts directory: %w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			metaFile := filepath.Join(contextDir, f.Name(), "meta.json")
			if _, err := os.Stat(metaFile); err == nil {
				if os.IsNotExist(err) {
					return envClient()
				}
				var dockerContext DockerContext
				content, err := os.ReadFile(metaFile)
				if err != nil {
					return nil, fmt.Errorf("unable to read Docker context meta file: %w", err)
				}
				if err := json.Unmarshal(content, &dockerContext); err != nil {
					return nil, fmt.Errorf("unable to parse Docker context meta file: %w", err)
				}
				if dockerContext.Name != config.CurrentContext {
					continue
				}

				endpoint, ok := dockerContext.Endpoints["docker"]
				if !ok {
					return nil, fmt.Errorf("docker endpoint not found in context")
				}

				return client.NewClientWithOpts(
					client.WithHost(endpoint.Host),
					client.WithAPIVersionNegotiation(),
				)
			}
		}
	}

	return envClient()
}
