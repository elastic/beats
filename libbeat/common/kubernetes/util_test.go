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
	// "context"
	// "fmt"
	// "io/ioutil"
	"os"
	//"github.com/pkg/errors"

	// "strings"
	"errors"
	"testing"

	//"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/client-go/kubernetes"
	// restclient "k8s.io/client-go/rest"
	// "k8s.io/client-go/tools/clientcmd"
	// clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	// "github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestDiscoverKubernetesNode(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	logger := logp.NewLogger("autodiscover.node")

	tests := []struct {
		host      string
		inCluster bool
		node      string
		err       error
		name      string
	}{
		{
			name:      "test value from config",
			host:      "worker-1",
			inCluster: false,
			node:      "worker-1",
			err:       nil,
		},
		{
			name:      "test value with env var",
			host:      "",
			inCluster: false,
			node:      "worker-2",
			err:       nil,
		},
		{
			name:      "test value with env var",
			host:      "",
			inCluster: false,
			node:      "worker-2",
			err:       nil,
		},
		{
			name:      "test value with env var not set",
			host:      "",
			inCluster: false,
			node:      "",
			err:       errors.New("kubernetes: Couldn't collect info from any of the files in /etc/machine-id /var/lib/dbus/machine-id: kubernetes: NODE_NAME environment variable was not set"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Unsetenv("NODE_NAME")
			if test.name == "test value with env var" {
				os.Setenv("NODE_NAME", "worker-2")
			}
			nodeName, error := DiscoverKubernetesNode(logger, test.host, test.inCluster, client)
			assert.Equal(t, test.node, nodeName)
			if error != nil {
				assert.Equal(t, test.err.Error(), error.Error())
			} else {
				assert.Equal(t, test.err, error)
			}
		})
	}
}
