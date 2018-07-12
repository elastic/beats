package skima

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestFlat(t *testing.T) {
	m := common.MapStr{
		"foo":    "bar",
		"baz":    1,
		"blargh": time.Duration(100),
	}

	validator := Schema(Map{
		"foo":    "bar",
		"baz":    1,
		"blargh": IsDuration,
	})

	validator(t, m)
}

func TestBadFlat(t *testing.T) {
	m := common.MapStr{}
	validator := Schema(Map{
		"notafield": IsDuration,
	})

	fakeT := new(testing.T)
	assert.Equal(fakeT, "foo", "bar")

	validator(fakeT, m)
	assert.True(t, fakeT.Failed())
}

func TestNested(t *testing.T) {
	m := common.MapStr{
		"foo": common.MapStr{
			"bar": "baz",
			"dur": time.Duration(100),
		},
	}

	validator := Schema(Map{
		"foo": Map{
			"bar": "baz",
		},
		"foo.dur": IsDuration,
	})

	validator(t, m)
}

func TestStrictFunc(t *testing.T) {
	m := common.MapStr{
		"foo": "bar",
		"baz": "bot",
	}

	validator := Schema(Map{
		"foo": "bar",
	})

	// Should pass, since this is not a strict check
	validator(t, m)

	fakeT := new(testing.T)
	Strict(validator)(fakeT, m)
	assert.True(t, fakeT.Failed())
}

func TestComposition(t *testing.T) {
	m := common.MapStr{
		"foo": "bar",
		"baz": "bot",
	}

	fooValidator := Schema(Map{"foo": "bar"})
	bazValidator := Schema(Map{"baz": "bot"})
	composed := Compose(fooValidator, bazValidator)

	fooValidator(t, m)
	bazValidator(t, m)
	composed(t, m)

	badValidator := Schema(Map{"notakey": "blah"})
	badComposed := Compose(badValidator, composed)

	fakeT := new(testing.T)
	badComposed(fakeT, m)
	assert.True(t, fakeT.Failed())
}

func TestComplex(t *testing.T) {
	m := common.MapStr{
		"foo": "bar",
		"hash": common.MapStr{
			"baz": 1,
			"bot": 2,
			"deep_hash": common.MapStr{
				"qux": "quark",
			},
		},
		"slice": []string{"pizza", "pasta", "and more"},
		"empty": nil,
	}

	validator := Schema(Map{
		"foo": "bar",
		"hash": StrictMap{
			"baz": 1,
			"bot": 2,
			"deep_hash": Map{
				"qux": "quark",
			},
		},
		"slice":        []string{"pizza", "pasta", "and more"},
		"empty":        DoesExist,
		"doesNotExist": DoesNotExist,
	})

	validator(t, m)

}
