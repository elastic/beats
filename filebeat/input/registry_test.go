package input

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/libbeat/common"
)

var fakeFactory = func(config *common.Config, outletFactory channel.Factory, context Context) (Input, error) {
	return nil, nil
}

func TestAddFactoryEmptyName(t *testing.T) {
	err := Register("", nil)
	if assert.Error(t, err) {
		assert.Equal(t, "Error registering input: name cannot be empty", err.Error())
	}
}

func TestAddNilFactory(t *testing.T) {
	err := Register("name", nil)
	if assert.Error(t, err) {
		assert.Equal(t, "Error registering input 'name': factory cannot be empty", err.Error())
	}
}

func TestAddFactoryTwice(t *testing.T) {
	var err error
	err = Register("name", fakeFactory)
	if err != nil {
		t.Fatal(err)
	}

	err = Register("name", fakeFactory)
	if assert.Error(t, err) {
		assert.Equal(t, "Error registering input 'name': already registered", err.Error())
	}
}

func TestGetFactory(t *testing.T) {
	f, err := GetFactory("name")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, f)
}

func TestGetNonExistentFactory(t *testing.T) {
	f, err := GetFactory("noSuchFactory")
	assert.Nil(t, f)
	if assert.Error(t, err) {
		assert.Equal(t, "Error creating input. No such input type exist: 'noSuchFactory'", err.Error())
	}
}
