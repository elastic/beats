package ucfg

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type myNonzeroInt int

func (m myNonzeroInt) Validate() error {
	if m == 0 {
		return errors.New("myNonzeroInt must not be 0")
	}
	return nil
}

func TestValidationPass(t *testing.T) {
	c, _ := NewFrom(map[string]interface{}{
		"a": 0,
		"b": 10,
		"d": -10,
		"f": 3.14,
	})

	tests := []interface{}{
		// validate field 'a'
		&struct {
			A int `validate:"positive"`
		}{},
		&struct {
			A int `validate:"positive,min=0"`
		}{},
		&struct {
			X int `config:"a" validate:"min=0"`
		}{},
		&struct {
			A time.Duration `validate:"positive"`
		}{},
		&struct {
			A time.Duration `validate:"positive,min=0"`
		}{},
		&struct {
			X time.Duration `config:"a" validate:"min=0"`
		}{},

		// validate field 'b'
		&struct {
			B int `validate:"nonzero"`
		}{},
		&struct {
			B myNonzeroInt
		}{},
		&struct {
			B int `validate:"positive"`
		}{},
		&struct {
			X int `config:"b" validate:"nonzero,min=-1"`
		}{},
		&struct {
			X int `config:"b" validate:"min=10, max=20"`
		}{},
		&struct {
			B time.Duration `validate:"nonzero"`
		}{},
		&struct {
			B time.Duration `validate:"positive"`
		}{},
		&struct {
			X time.Duration `config:"b" validate:"min=10, max=20"`
		}{},
		&struct {
			X time.Duration `config:"b" validate:"min=10s, max=20s"`
		}{},

		// validate field 'd'
		&struct {
			D int `validate:"nonzero"`
		}{},
		&struct {
			X int `config:"d" validate:"nonzero,min=-10"`
		}{},
		&struct {
			X int `config:"d" validate:"min=-10, max=0"`
		}{},
		&struct {
			D time.Duration `validate:"nonzero"`
		}{},
		&struct {
			X time.Duration `config:"d" validate:"nonzero,min=-10"`
		}{},
		&struct {
			X time.Duration `config:"d" validate:"min=-10, max=0"`
		}{},

		// validate field 'f'
		&struct {
			F float64 `validate:"nonzero"`
		}{},
		&struct {
			F float64 `validate:"positive"`
		}{},
		&struct {
			X int `config:"f" validate:"nonzero,min=-1"`
		}{},
		&struct {
			X int `config:"f" validate:"min=3, max=20"`
		}{},
		&struct {
			F time.Duration `validate:"nonzero"`
		}{},
		&struct {
			F time.Duration `validate:"positive"`
		}{},
		&struct {
			X time.Duration `config:"f" validate:"nonzero,min=-1"`
		}{},
		&struct {
			X time.Duration `config:"f" validate:"min=3, max=20"`
		}{},

		// other
		&struct {
			X int // field not present in config, but not required
		}{},
	}

	for i, test := range tests {
		t.Logf("Test config (%v): %#v", i, test)

		err := c.Unpack(test)
		assert.NoError(t, err)
	}
}

func TestValidationFail(t *testing.T) {
	c, _ := NewFrom(map[string]interface{}{
		"a": 0,
		"b": 10,
		"d": -10,
		"f": 3.14,
	})

	tests := []interface{}{
		// test field 'a'
		&struct {
			X int `config:"a" validate:"nonzero"`
		}{},
		&struct {
			X myNonzeroInt `config:"a"`
		}{},
		&struct {
			X int `config:"a" validate:"min=10"`
		}{},
		&struct {
			X time.Duration `config:"a" validate:"nonzero"`
		}{},
		&struct {
			X time.Duration `config:"a" validate:"min=10"`
		}{},
		&struct {
			X time.Duration `config:"a" validate:"min=10ns"`
		}{},

		// test field 'b'
		&struct {
			X int `config:"b" validate:"max=8"`
		}{},
		&struct {
			X int `config:"b" validate:"min=20"`
		}{},
		&struct {
			X time.Duration `config:"b" validate:"max=8ms"`
		}{},
		&struct {
			X time.Duration `config:"b" validate:"min=20h"`
		}{},

		// test field 'd'
		&struct {
			X int `config:"d" validate:"positive"`
		}{},
		&struct {
			X int `config:"d" validate:"max=-11"`
		}{},
		&struct {
			X int `config:"d" validate:"min=20"`
		}{},
		&struct {
			X time.Duration `config:"d" validate:"positive"`
		}{},
		&struct {
			X time.Duration `config:"d" validate:"max=-11s"`
		}{},
		&struct {
			X time.Duration `config:"d" validate:"min=20h"`
		}{},

		// test field 'f'
		&struct {
			X float64 `config:"f" validate:"max=1"`
		}{},
		&struct {
			X float64 `config:"f" validate:"min=20"`
		}{},
		&struct {
			X time.Duration `config:"f" validate:"max=1s"`
		}{},
		&struct {
			X time.Duration `config:"f" validate:"min=20s"`
		}{},

		// other
		&struct {
			X int `validate:"required"`
		}{},
	}

	for i, test := range tests {
		t.Logf("Test config (%v): %#v", i, test)

		err := c.Unpack(test)
		assert.True(t, err != nil)
	}
}
