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
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/keystore"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// KubernetesKeystoresRegistry implements a Provider for Keystore.
type KubernetesKeystoresRegistry struct {
	logger *logp.Logger
	client k8s.Interface
}

// KubernetesSecretsKeystore allows to retrieve passwords from Kubernetes secrets for a given namespace
type KubernetesSecretsKeystore struct {
	namespace string
	client    k8s.Interface
	logger    *logp.Logger
}

// Factoryk8s Create the right keystore with the configured options
func Factoryk8s(keystoreNamespace string, ks8client k8s.Interface, logger *logp.Logger) (keystore.Keystore, error) {
	keystore, err := NewKubernetesSecretsKeystore(keystoreNamespace, ks8client, logger)
	return keystore, err
}

// NewKubernetesKeystoresRegistry initializes a KubernetesKeystoresRegistry
func NewKubernetesKeystoresRegistry(logger *logp.Logger, client k8s.Interface) keystore.Provider {
	return &KubernetesKeystoresRegistry{
		logger: logger,
		client: client,
	}
}

// GetKeystore return a KubernetesSecretsKeystore if it already exists for a given namespace or creates a new one.
func (kr *KubernetesKeystoresRegistry) GetKeystore(event bus.Event) keystore.Keystore {
	namespace := ""
	if val, ok := event["kubernetes"]; ok {
		kubernetesMeta := val.(common.MapStr)
		ns, err := kubernetesMeta.GetValue("namespace")
		if err != nil {
			kr.logger.Debugf("Cannot retrieve kubernetes namespace from event: %s", event)
			return nil
		}
		namespace = ns.(string)
	}
	if namespace != "" {
		k8sKeystore, _ := Factoryk8s(namespace, kr.client, kr.logger)
		return k8sKeystore
	}
	kr.logger.Debugf("Cannot retrieve kubernetes namespace from event: %s", event)
	return nil
}

// NewKubernetesSecretsKeystore returns an new k8s Keystore
func NewKubernetesSecretsKeystore(keystoreNamespace string, ks8client k8s.Interface, logger *logp.Logger) (keystore.Keystore, error) {
	keystore := KubernetesSecretsKeystore{
		namespace: keystoreNamespace,
		client:    ks8client,
		logger:    logger,
	}
	return &keystore, nil
}

// Retrieve return a SecureString instance that will contains both the key and the secret.
func (k *KubernetesSecretsKeystore) Retrieve(key string) (*keystore.SecureString, error) {
	// key = "kubernetes.somenamespace.somesecret.value"
	tokens := strings.Split(key, ".")
	if len(tokens) > 0 && tokens[0] != "kubernetes" {
		return nil, keystore.ErrKeyDoesntExists
	}
	if len(tokens) != 4 {
		k.logger.Debugf(
			"not valid secret key: %v. Secrets should be of the following format %v",
			key,
			"kubernetes.somenamespace.somesecret.value",
		)
		return nil, keystore.ErrKeyDoesntExists
	}
	ns := tokens[1]
	secretName := tokens[2]
	secretVar := tokens[3]
	if ns != k.namespace {
		k.logger.Debugf("cannot access Kubernetes secrets from a different namespace (%v) than: %v", ns, k.namespace)
		return nil, keystore.ErrKeyDoesntExists
	}
	secretIntefrace := k.client.CoreV1().Secrets(ns)
	ctx := context.TODO()
	secret, err := secretIntefrace.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		k.logger.Errorf("Could not retrieve secret from k8s API: %v", err)
		return nil, keystore.ErrKeyDoesntExists
	}
	if _, ok := secret.Data[secretVar]; !ok {
		k.logger.Errorf("Could not retrieve value %v for secret %v", secretVar, secretName)
		return nil, keystore.ErrKeyDoesntExists
	}
	secretString := secret.Data[secretVar]
	return keystore.NewSecureString(secretString), nil
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
