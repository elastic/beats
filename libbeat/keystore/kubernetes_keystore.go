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

package keystore

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/libbeat/common"
)

type KubernetesKeystores map[string]KubernetesSecretsKeystore

var kubernetesKeystoresRegistry = KubernetesKeystores{}

// KubernetesSecretsKeystore Allows to retrieve passwords from Kubernetes secrets for a given namespace.
type KubernetesSecretsKeystore struct {
	namespace string
	client    k8s.Interface
}

// NewKubernetesSecretsKeystore returns an new k8s Keystore
func NewKubernetesSecretsKeystore(keystoreNamespace string, ks8client k8s.Interface) (Keystore, error) {
	// search for in the keystore registry to find if there is keystore already for that namespace
	storedKeystore, err := lookupForKeystore(keystoreNamespace)
	if err != nil {
		return nil, err
	}
	if storedKeystore != nil {
		return storedKeystore, nil
	}

	keystore := KubernetesSecretsKeystore{
		namespace: keystoreNamespace,
		client:    ks8client,
	}
	kubernetesKeystoresRegistry[keystoreNamespace] = keystore
	return &keystore, nil
}

func lookupForKeystore(keystoreNamespace string) (Keystore, error) {
	if keystore, ok := kubernetesKeystoresRegistry[keystoreNamespace]; ok {
		return &keystore, nil
	}
	return nil, nil
}

// Retrieve return a SecureString instance that will contains both the key and the secret.
func (k *KubernetesSecretsKeystore) Retrieve(key string) (*SecureString, error) {
	// key = "kubernetes:somenamespace:somesecret:value"
	toks := strings.Split(key, ":")
	ns := toks[1]
	secretName := toks[2]
	secretVar := toks[3]
	if ns != k.namespace {
		return nil, fmt.Errorf("cannot access Kubernetes secrets from a different namespace than: %v", ns)
	}
	secret, err := k.client.CoreV1().Secrets(ns).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	secretString := secret.Data[secretVar]
	return NewSecureString(secretString), nil
}

// GetConfig returns common.Config representation of the key / secret pair to be merged with other
// loaded configuration.
func (k *KubernetesSecretsKeystore) GetConfig() (*common.Config, error) {
	return nil, nil
}

// IsPersisted return if the keystore is physically persisted on disk.
func (k *KubernetesSecretsKeystore) IsPersisted() bool {
	return true
}
