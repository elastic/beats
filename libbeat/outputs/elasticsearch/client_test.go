// +build !integration

package elasticsearch

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/stretchr/testify/assert"
)

func readStatusItem(in []byte) (int, string, error) {
	reader := newJSONReader(in)
	code, msg, err := itemStatus(reader)
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
	response := []byte(`{"create": {"status": 400, "error": "test error"}}`)
	code, msg, err := readStatusItem(response)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, `"test error"`, msg)
}

func TestES2StyleErrorStatus(t *testing.T) {
	response := []byte(`{"create": {"status": 400, "error": {"reason": "test_error"}}}`)
	code, msg, err := readStatusItem(response)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, `{"reason": "test_error"}`, msg)
}

func TestES2StyleExtendedErrorStatus(t *testing.T) {
	response := []byte(`
    {
      "create": {
        "status": 400,
        "error": {
          "reason": "test_error",
          "transient": false,
          "extra": null
        }
      }
    }`)
	code, _, err := readStatusItem(response)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
}

func TestCollectPublishFailsNone(t *testing.T) {
	N := 100
	item := `{"create": {"status": 200}},`
	response := []byte(`{"items": [` + strings.Repeat(item, N) + `]}`)

	event := common.MapStr{"field": 1}
	events := make([]common.MapStr, N)
	for i := 0; i < N; i++ {
		events[i] = event
	}

	reader := newJSONReader(response)
	res := bulkCollectPublishFails(reader, events)
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

	reader := newJSONReader(response)
	res := bulkCollectPublishFails(reader, events)
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

	reader := newJSONReader(response)
	res := bulkCollectPublishFails(reader, events)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, events, res)
}

func TestGetIndexStandard(t *testing.T) {

	time := time.Now().UTC()
	extension := fmt.Sprintf("%d.%02d.%02d", time.Year(), time.Month(), time.Day())

	event := common.MapStr{
		"@timestamp": common.Time(time),
		"field":      1,
	}

	pattern := "beatname-%{+yyyy.MM.dd}"
	fmtstr := fmtstr.MustCompileEvent(pattern)
	indexSel := outil.MakeSelector(outil.FmtSelectorExpr(fmtstr, ""))

	index := getIndex(event, indexSel)
	assert.Equal(t, index, "beatname-"+extension)
}

func TestGetIndexOverwrite(t *testing.T) {

	time := time.Now().UTC()
	extension := fmt.Sprintf("%d.%02d.%02d", time.Year(), time.Month(), time.Day())

	event := common.MapStr{
		"@timestamp": common.Time(time),
		"field":      1,
		"beat": common.MapStr{
			"name":  "testbeat",
			"index": "dynamicindex",
		},
	}

	pattern := "beatname-%%{+yyyy.MM.dd}"
	fmtstr := fmtstr.MustCompileEvent(pattern)
	indexSel := outil.MakeSelector(outil.FmtSelectorExpr(fmtstr, ""))

	index := getIndex(event, indexSel)
	assert.Equal(t, index, "dynamicindex-"+extension)
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

	reader := newJSONReader(nil)
	for i := 0; i < b.N; i++ {
		reader.init(response)
		res := bulkCollectPublishFails(reader, events)
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

	reader := newJSONReader(nil)
	for i := 0; i < b.N; i++ {
		reader.init(response)
		res := bulkCollectPublishFails(reader, events)
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

	reader := newJSONReader(nil)
	for i := 0; i < b.N; i++ {
		reader.init(response)
		res := bulkCollectPublishFails(reader, events)
		if len(res) != 3 {
			b.Fail()
		}
	}
}
