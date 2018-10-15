package qcgen

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing/quick"
)

type Generator struct {
	arguments []userGen
}

type userGen func(rng *rand.Rand, params []reflect.Value) reflect.Value

var tRand = reflect.TypeOf((*rand.Rand)(nil))

// NewGenerator creates a new generator. Each function must implement
// `func(*rand.Rand) T`, with T being the custom type to be generated.
// The generators Gen methods selects the function to execute on
// matching return type.
func NewGenerator(testFn interface{}, fns ...interface{}) *Generator {
	mapping := map[reflect.Type]reflect.Value{}

	for i, fn := range fns {
		v := reflect.ValueOf(fn)
		t := v.Type()
		if t.Kind() != reflect.Func {
			panic(fmt.Errorf("argument %v is no function", i))
		}

		if t.NumIn() != 1 || t.NumOut() != 1 {
			panic(fmt.Errorf("argument %v must accept one argument and return one value", i))
		}

		tIn := t.In(0)
		if tIn != tRand {
			panic(fmt.Errorf("argument %v must accept *rand.Rand as input only", i))
		}

		mapping[t.Out(0)] = v
	}

	fn := reflect.TypeOf(testFn)
	argGen := make([]userGen, fn.NumIn())
	for i := range argGen {
		tIn := fn.In(i)
		if v, exists := mapping[tIn]; exists {
			argGen[i] = makeUserGen(v)
		} else {
			argGen[i] = makeDefaultGen(tIn)
		}
	}

	return &Generator{argGen}
}

func makeUserGen(fn reflect.Value) userGen {
	return func(_ *rand.Rand, params []reflect.Value) reflect.Value {
		out := fn.Call(params)
		return out[0]
	}
}

func makeDefaultGen(t reflect.Type) userGen {
	return func(rng *rand.Rand, _ []reflect.Value) reflect.Value {
		out, ok := quick.Value(t, rng)
		if !ok {
			panic(fmt.Errorf("cannot create arbitrary value of type %s", t))
		}
		return out
	}
}

func (g *Generator) Gen(args []reflect.Value, rng *rand.Rand) {
	rngParam := []reflect.Value{reflect.ValueOf(rng)}
	for i := range args {
		args[i] = g.arguments[i](rng, rngParam)
	}
}
