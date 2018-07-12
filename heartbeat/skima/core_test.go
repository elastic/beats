package skima

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"time"

	"github.com/elastic/beats/libbeat/common"
)

func TestFlat(t *testing.T) {
	m := common.MapStr{
		"foo": "bar",
		"baz": 1,
	}

	results := Schema(Map{
		"foo": "bar",
		"baz": IsIntGt(0),
	})(m)

	Test(t, results)
}

func TestBadFlat(t *testing.T) {
	m := common.MapStr{}

	fakeT := new(testing.T)

	results := Schema(Map{
		"notafield": IsDuration,
	})(m)

	Test(fakeT, results)
	assert.True(t, fakeT.Failed())

	result := results["notafield"][0]
	assert.False(t, result.valid)
	assert.Contains(t, result.message, "Expected a time.duration")
}

func TestNested(t *testing.T) {
	m := common.MapStr{
		"foo": common.MapStr{
			"bar": "baz",
			"dur": time.Duration(100),
		},
	}

	results := Schema(Map{
		"foo": Map{
			"bar": "baz",
		},
		"foo.dur": IsDuration,
	})(m)

	Test(t, results)

	assert.Len(t, results, 3, "One result per matcher")
}

/*
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

}*/
