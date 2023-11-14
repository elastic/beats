package network

import (
	"context"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/require"
)

func TestNetworkTrack(t *testing.T) {
	_ = logp.DevelopmentSetup(logp.WithLevel(logp.InfoLevel))
	tracker, err := NewNetworkTracker()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	tracker.Track(ctx)

	time.Sleep(time.Minute * 10)
}
