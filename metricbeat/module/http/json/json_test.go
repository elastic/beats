package json

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/assert"
)

func TestEventMapper(t *testing.T) {
	var actualJSONBody map[string]interface{}
	var expectedJSONBody map[string]interface{}

	absPath, err := filepath.Abs("./_meta/test")
	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	actualJSONResponse, err := ioutil.ReadFile(absPath + "/json_response_with_dots.json")
	assert.Nil(t, err)
	err = json.Unmarshal(actualJSONResponse, &actualJSONBody)
	assert.Nil(t, err)

	dedottedJSONResponse, err := ioutil.ReadFile(absPath + "/json_response_dedot.json")
	assert.Nil(t, err)
	err = json.Unmarshal(dedottedJSONResponse, &expectedJSONBody)
	assert.Nil(t, err)

	actualJSONBody = replaceDots(actualJSONBody).(map[string]interface{})

	assert.Equal(t, expectedJSONBody, actualJSONBody)
}
