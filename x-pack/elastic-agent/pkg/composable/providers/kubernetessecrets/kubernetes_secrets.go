// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetessecrets

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	corecomp "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

var _ corecomp.FetchContextProvider = (*contextProviderK8sSecrets)(nil)
var getK8sClientFunc = getK8sClient

func init() {
	composable.Providers.AddContextProvider("kubernetes_secrets", ContextProviderBuilder)
}

type contextProviderK8sSecrets struct {
	logger *logger.Logger
	config *Config

	client k8sclient.Interface
}

// ContextProviderBuilder builds the context provider.
func ContextProviderBuilder(logger *logger.Logger, c *config.Config) (corecomp.ContextProvider, error) {
	var cfg Config
	if c == nil {
		c = config.New()
	}
	err := c.Unpack(&cfg)
	if err != nil {
		return nil, errors.New(err, "failed to unpack configuration")
	}
	return &contextProviderK8sSecrets{logger, &cfg, nil}, nil
}

func (p *contextProviderK8sSecrets) Fetch(key string) (string, bool) {
	// key = "kubernetes_secrets.somenamespace.somesecret.value"
	if p.client == nil {
		return "", false
	}
	tokens := strings.Split(key, ".")
	if len(tokens) > 0 && tokens[0] != "kubernetes_secrets" {
		return "", false
	}
	if len(tokens) != 4 {
		p.logger.Debugf(
			"not valid secret key: %v. Secrets should be of the following format %v",
			key,
			"kubernetes_secrets.somenamespace.somesecret.value",
		)
		return "", false
	}
	ns := tokens[1]
	secretName := tokens[2]
	secretVar := tokens[3]

	secretIntefrace := p.client.CoreV1().Secrets(ns)
	ctx := context.TODO()
	secret, err := secretIntefrace.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		p.logger.Errorf("Could not retrieve secret from k8s API: %v", err)
		return "", false
	}
	if _, ok := secret.Data[secretVar]; !ok {
		p.logger.Errorf("Could not retrieve value %v for secret %v", secretVar, secretName)
		return "", false
	}
	secretString := secret.Data[secretVar]
	return string(secretString), true
}

// Run initializes the k8s secrets context provider.
func (p *contextProviderK8sSecrets) Run(comm corecomp.ContextProviderComm) error {
	client, err := getK8sClientFunc(p.config.KubeConfig, p.config.KubeClientOptions)
	if err != nil {
		p.logger.Debugf("Kubernetes_secrets provider skipped, unable to connect: %s", err)
		return nil
	}
	p.client = client
	return nil
}

func getK8sClient(kubeconfig string, opt kubernetes.KubeClientOptions) (k8sclient.Interface, error) {
	return kubernetes.GetKubernetesClient(kubeconfig, opt)
}
