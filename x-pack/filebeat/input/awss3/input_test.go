// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetProviderFromDomain(t *testing.T) {
	assert.Equal(t, "aws", getProviderFromDomain("", ""))
	assert.Equal(t, "aws", getProviderFromDomain("c2s.ic.gov", ""))
	assert.Equal(t, "abc", getProviderFromDomain("abc.com", "abc"))
	assert.Equal(t, "xyz", getProviderFromDomain("oraclecloud.com", "xyz"))
	assert.Equal(t, "aws", getProviderFromDomain("amazonaws.com", ""))
	assert.Equal(t, "aws", getProviderFromDomain("c2s.sgov.gov", ""))
	assert.Equal(t, "aws", getProviderFromDomain("c2s.ic.gov", ""))
	assert.Equal(t, "aws", getProviderFromDomain("amazonaws.com.cn", ""))
	assert.Equal(t, "backblaze", getProviderFromDomain("https://backblazeb2.com", ""))
	assert.Equal(t, "wasabi", getProviderFromDomain("https://wasabisys.com", ""))
	assert.Equal(t, "digitalocean", getProviderFromDomain("https://digitaloceanspaces.com", ""))
	assert.Equal(t, "dreamhost", getProviderFromDomain("https://dream.io", ""))
	assert.Equal(t, "scaleway", getProviderFromDomain("https://scw.cloud", ""))
	assert.Equal(t, "gcp", getProviderFromDomain("https://googleapis.com", ""))
	assert.Equal(t, "arubacloud", getProviderFromDomain("https://cloud.it", ""))
	assert.Equal(t, "linode", getProviderFromDomain("https://linodeobjects.com", ""))
	assert.Equal(t, "vultr", getProviderFromDomain("https://vultrobjects.com", ""))
	assert.Equal(t, "ibm", getProviderFromDomain("https://appdomain.cloud", ""))
	assert.Equal(t, "alibaba", getProviderFromDomain("https://aliyuncs.com", ""))
	assert.Equal(t, "oracle", getProviderFromDomain("https://oraclecloud.com", ""))
	assert.Equal(t, "exoscale", getProviderFromDomain("https://exo.io", ""))
	assert.Equal(t, "upcloud", getProviderFromDomain("https://upcloudobjects.com", ""))
	assert.Equal(t, "iland", getProviderFromDomain("https://ilandcloud.com", ""))
	assert.Equal(t, "zadara", getProviderFromDomain("https://zadarazios.com", ""))
}
