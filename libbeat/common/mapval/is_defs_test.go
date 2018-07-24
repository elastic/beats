package mapval

import (
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
)

func assertIsDefValid(t *testing.T, id IsDef, value interface{}) {
	res := id.check(value, true)
	if !res.Valid {
		assert.Fail(
			t,
			"Expected Valid IsDef",
			"Isdef %#v was not valid for value %#v with error: ", id, value, res.Message,
		)
	}
}

func assertIsDefInvalid(t *testing.T, id IsDef, value interface{}) {
	res := id.check(value, true)
	if res.Valid {
		assert.Fail(
			t,
			"Expected invalid IsDef",
			"Isdef %#v was should not have been valid for value %#v",
			id,
			value,
		)
	}
}

func TestIsAny(t *testing.T) {
	id := IsAny(IsEqual("foo"), IsEqual("bar"))

	assertIsDefValid(t, id, "foo")
	assertIsDefValid(t, id, "bar")
	assertIsDefInvalid(t, id, "basta")
}

func TestIsEqual(t *testing.T) {
	id := IsEqual("foo")

	assertIsDefValid(t, id, "foo")
	assertIsDefInvalid(t, id, "bar")
}

func TestIsStringContaining(t *testing.T) {
	id := IsStringContaining("foo")

	assertIsDefValid(t, id, "foo")
	assertIsDefValid(t, id, "a foo b")
	assertIsDefInvalid(t, id, "a bar b")
}

func TestIsDuration(t *testing.T) {
	id := IsDuration

	assertIsDefValid(t, id, time.Duration(1))
	assertIsDefInvalid(t, id, "foo")
}

func TestIsIntGt(t *testing.T) {
	id := IsIntGt(100)

	assertIsDefValid(t, id, 101)
	assertIsDefInvalid(t, id, 100)
	assertIsDefInvalid(t, id, 99)
}

func TestIsNil(t *testing.T) {
	assertIsDefValid(t, IsNil, nil)
	assertIsDefInvalid(t, IsNil, "foo")
}
