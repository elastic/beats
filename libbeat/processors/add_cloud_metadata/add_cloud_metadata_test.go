package add_cloud_metadata

import (
	"context"
	"testing"
	"time"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_addCloudMetadata_String(t *testing.T) {
	const deadline = 100 * time.Millisecond
	ctx, cancel := context.WithTimeout(t.Context(), deadline)
	defer cancel()
	cfg := conf.MustNewConfigFrom(map[string]any{
		"providers": []string{"openstack"},
		"host":      "fake:1234",
		"timeout":   (2 * deadline).String(),
	})
	p, err := New(cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	assert.Contains(t, p.String(), "add_cloud_metadata=<uninitialized>")
	require.NoError(t, ctx.Err())

	time.Sleep(3 * deadline)
	assert.Contains(t, p.String(), "add_cloud_metadata={}")
}
