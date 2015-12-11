package elasticsearch

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func readStatusItem(in []byte) (int, string, error) {
	code, msg, err := itemStatus(in)
	return code, string(msg), err
}

func TestESNoErrorStatus(t *testing.T) {
	response := []byte(`{"create": {"status": 200}}`)
	code, msg, err := readStatusItem(response)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Equal(t, "", msg)
}

func TestES1StyleErrorStatus(t *testing.T) {
	response := json.RawMessage(`{"create": {"status": 400, "error": "test error"}}`)
	code, msg, err := readStatusItem(response)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, `"test error"`, msg)
}

func TestES2StyleErrorStatus(t *testing.T) {
	response := json.RawMessage(`{"create": {"status": 400, "error": {"reason": "test_error"}}}`)
	code, msg, err := readStatusItem(response)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, `{"reason": "test_error"}`, msg)
}

func TestCollectPublishFailsNone(t *testing.T) {
	N := 100
	item := `{"create": {"status": 200}},`
	items := strings.Repeat(item, N)
	response := []byte(`{"items": [` + items[:len(items)-1] + `]}`)

	event := common.MapStr{"field": 1}
	events := make([]common.MapStr, N)
	for i := 0; i < N; i++ {
		events[i] = event
	}

	bs, _ := readBulkResult(response)
	res := bulkCollectPublishFails(bs, events)
	assert.Equal(t, 0, len(res))
}

func TestCollectPublishFailMiddle(t *testing.T) {
	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 200}}
    ]}
  `)

	event := common.MapStr{"field": 1}
	eventFail := common.MapStr{"field": 2}
	events := []common.MapStr{event, eventFail, event}

	bs, _ := readBulkResult(response)
	res := bulkCollectPublishFails(bs, events)
	assert.Equal(t, 1, len(res))
	if len(res) == 1 {
		assert.Equal(t, eventFail, res[0])
	}
}

func TestCollectPublishFailAll(t *testing.T) {
	response := []byte(`
    { "items": [
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 429, "error": "ups"}}
    ]}
  `)

	event := common.MapStr{"field": 2}
	events := []common.MapStr{event, event, event}

	bs, _ := readBulkResult(response)
	res := bulkCollectPublishFails(bs, events)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, events, res)
}

func BenchmarkCollectPublishFailsNone(b *testing.B) {
	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {"status": 200}},
      {"create": {"status": 200}}
    ]}
  `)

	event := common.MapStr{"field": 1}
	events := []common.MapStr{event, event, event}
	bs, err := readBulkResult(response)
	if err != nil {
		b.Fatalf("test setup failed with: %v", err)
	}

	for i := 0; i < b.N; i++ {
		res := bulkCollectPublishFails(bs, events)
		if len(res) != 0 {
			b.Fail()
		}
	}
}

func BenchmarkCollectPublishFailMiddle(b *testing.B) {
	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 200}}
    ]}
  `)

	event := common.MapStr{"field": 1}
	eventFail := common.MapStr{"field": 2}
	events := []common.MapStr{event, eventFail, event}
	bs, _ := readBulkResult(response)

	for i := 0; i < b.N; i++ {
		res := bulkCollectPublishFails(bs, events)
		if len(res) != 1 {
			b.Fail()
		}
	}
}

func BenchmarkCollectPublishFailAll(b *testing.B) {
	response := []byte(`
    { "items": [
      {"creatMiddlee": {"status": 429, "error": "ups"}},
      {"creatMiddlee": {"status": 429, "error": "ups"}},
      {"creatMiddlee": {"status": 429, "error": "ups"}}
    ]}
  `)

	event := common.MapStr{"field": 2}
	events := []common.MapStr{event, event, event}
	bs, _ := readBulkResult(response)

	for i := 0; i < b.N; i++ {
		res := bulkCollectPublishFails(bs, events)
		if len(res) != 3 {
			b.Fail()
		}
	}
}
