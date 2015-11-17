package elasticsearch

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestESNoErrorStatus(t *testing.T) {
	response := json.RawMessage(`{"create": {"status": 200}}`)
	code, msg, err := itemStatus(response)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Equal(t, "", msg)
}

func TestES1StyleErrorStatus(t *testing.T) {
	response := json.RawMessage(`{"create": {"status": 400, "error": "test error"}}`)
	code, msg, err := itemStatus(response)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, `"test error"`, msg)
}

func TestES2StyleErrorStatus(t *testing.T) {
	response := json.RawMessage(`{"create": {"status": 400, "error": {"reason": "test_error"}}}`)
	code, msg, err := itemStatus(response)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, `{"reason": "test_error"}`, msg)
}
