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
	tc := newESTestContext(t)

	m1idUUID, _ := uuid.NewV4()
	m1ID := m1idUUID.String()
	m1Typ := "testtyp"
	initState := newMonitorState(m1ID, StatusUp)
	tc.setInitialState(t, m1Typ, initState)
	ms := tc.tracker.RecordStatus(m1ID, StatusUp)
	require.Equal(t, 2, ms.Checks)
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
