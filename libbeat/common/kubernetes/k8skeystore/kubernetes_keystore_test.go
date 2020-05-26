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

package k8skeystore

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
)

func TestGetKeystore(t *testing.T) {
	kRegistry := NewKubernetesKeystoresRegistry(nil, nil)
	k1 := kRegistry.GetKeystore(bus.Event{"kubernetes": common.MapStr{"namespace": "my_namespace"}})
	k2 := kRegistry.GetKeystore(bus.Event{"kubernetes": common.MapStr{"namespace": "my_namespace"}})
	assert.Equal(t, k1, k2)
	k3 := kRegistry.GetKeystore(bus.Event{"kubernetes": common.MapStr{"namespace": "my_namespace_2"}})
	assert.NotEqual(t, k2, k3)
}

// TODO: upgrade client dependency and use fake client to test retrieve
//func TestGetKeystoreAndRetrieve(t *testing.T) {
//	client := k8sfake.NewSimpleClientset()
//	ns := "test_namespace"
//	pass := "testing_passpass"
//	secret := &v1.Secret{
//		TypeMeta: metav1.TypeMeta{
//			Kind:       "Secret",
//			APIVersion: "apps/v1beta1",
//		},
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "testing_secret",
//			Namespace: ns,
//		},
//		Data: map[string][]byte{
//			"secret_value": []byte(pass),
//		},
//	}
//	client.CoreV1().Secrets(ns).Create(context.TODO(), secret, metav1.CreateOptions{})
//
//	kRegistry := NewKubernetesKeystoresRegistry(nil, nil)
//	k1 := kRegistry.GetKeystore(bus.Event{"kubernetes": common.MapStr{"namespace": ns}})
//	key := "kubernetes.test_namespace.testing_secret.secret_value"
//	secretVal, err := k1.Retrieve(key)
//	if err != nil {
//		t.Fatalf("could not retrive k8s secret", err)
//	}
//	assert.Equal(t, pass, secretVal)
//}
