package monitorstate

import (
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/heartbeat/esutil"
)

func TestStates(t *testing.T) {
	etc := newESTestContext(t)

	// Create three monitors in ES, load their states, and make sure we track them correctly
	// We create 3 to make sure the query isolates the monitors correctly
	for i := 0; i < 3; i++ {
		monID := etc.createTestMonitorStateInES(t, StatusUp)
		// Since we've continued this state it should register the initial state
		ms := etc.tracker.getCurrentState(monID)
		requireMSCounts(t, ms, 1, 0)

		_ = etc.tracker.RecordStatus(monID, StatusUp)
		ms = etc.tracker.RecordStatus(monID, StatusUp)
		requireMSCounts(t, ms, 3, 0)
	}

	// Let's test a final one with a down state for completeness
	monID := etc.createTestMonitorStateInES(t, StatusDown)
	_ = etc.tracker.RecordStatus(monID, StatusDown)
	_ = etc.tracker.RecordStatus(monID, StatusDown)
	_ = etc.tracker.RecordStatus(monID, StatusDown)
	ms := etc.tracker.RecordStatus(monID, StatusDown)
	requireMSCounts(t, ms, 0, 3)
}

type esTestContext struct {
	namespace string
	esc       *elasticsearch.Client
	loader    StateLoader
	tracker   *MonitorStateTracker
}

func newESTestContext(t *testing.T) *esTestContext {
	namespace, _ := uuid.NewV4()
	esc := integES(t)
	etc := &esTestContext{
		namespace: namespace.String(),
		esc:       esc,
		loader:    MakeESLoader(esc, fmt.Sprintf("synthetics-*-%s", namespace.String())),
	}

	etc.tracker = NewMonitorStateTracker(etc.loader)

	return etc
}

func (etc *esTestContext) createTestMonitorStateInES(t *testing.T, s StateStatus) (id string) {
	mUUID, _ := uuid.NewV4()
	mID := mUUID.String()
	mType := "testtyp"
	initState := newMonitorState(mID, s)
	etc.setInitialState(t, mType, initState)
	return mID
}

func (etc *esTestContext) setInitialState(t *testing.T, typ string, ms *MonitorState) {
	idx := fmt.Sprintf("synthetics-%s-%s", typ, etc.namespace)

	type Mon struct {
		Id   string `json:"id"`
		Type string `json:"type"`
	}

	reqBodyRdr, err := esutil.ToJsonRdr(struct {
		Ts      time.Time     `json:"@timestamp"`
		Monitor Mon           `json:"monitor"`
		State   *MonitorState `json:"state"`
	}{
		Ts:      time.Now(),
		Monitor: Mon{Id: ms.MonitorId, Type: typ},
		State:   ms,
	})

	_, err = esutil.CheckRetResp(etc.esc.Index(idx, reqBodyRdr, func(request *esapi.IndexRequest) {
		// Refresh the index since we tend to re-query immediately, otherwise this would miss
		request.Refresh = "true"

	}))
	require.NoError(t, err)
}

var connOnce = &sync.Once{}

func integES(t *testing.T) (esc *elasticsearch.Client) {
	connOnce.Do(func() {
		var err error
		esc, err = elasticsearch.NewClient(elasticsearch.Config{
			Addresses: []string{"http://127.0.0.1:9200"},
			Username:  "admin",
			Password:  "testing",
		})
		require.NoError(t, err)
		respBody, err := esc.Cluster.Health()
		healthRaw, err := esutil.CheckRetResp(respBody, err)
		require.NoError(t, err)

		healthResp := struct {
			Status string `json:"status"`
		}{}
		err = json.Unmarshal(healthRaw, &healthResp)
		require.NoError(t, err)
		require.Contains(t, []string{"green", "yellow"}, healthResp.Status)
	})

	return esc
}
