package memory

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/elastic/beats/v7/libbeat/opt"
)

type CustomFloat interface {
	IsZero() bool
}

type ZeroTest struct {
	Zero bool
}

func (z ZeroTest) MarshalJSON() ([]byte, error) {
	return json.Marshal(z.Zero)
}

func (z ZeroTest) IsZero() bool {
	return z.Zero
}

type UsedMemStatsTest struct {
	Raw   float64     `json:"raw,omitempty"`
	Iface opt.OptType `json:"iface,omitempty"`
}

// func (s UsedMemStatsTest) MarshalJSON() ([]byte, error) {

// 	type sAlias UsedMemStatsTest

// 	if s.Iface.IsZero() {
// 		s.Iface = nil
// 	}

// 	return json.Marshal(sAlias(s))
// }

type MarshalWrapper struct {
	Butterfly interface{}
}

func (m MarshalWrapper) MarshalJSON() ([]byte, error) {

	bv := reflect.ValueOf(m.Butterfly).Elem()
	for i := 0; i < bv.NumField(); i++ {
		if bv.Field(i).CanInterface() {
			fiface := bv.Field(i).Interface()
			zeroIface, ok := fiface.(CustomFloat)
			if ok {
				if zeroIface.IsZero() {
					zeroField := reflect.ValueOf(m.Butterfly).Elem().Field(i)
					fmt.Printf("===%v\n", zeroField.Type())
					if zeroField.CanSet() {
						zeroField.Set(reflect.Zero(zeroField.Type()))
					} else {
						fmt.Printf("Can't Set field %v\n", zeroField.Type())
					}
				}

			}
		}
	}
	return json.Marshal(m.Butterfly)
}

func runJsonMarshal(input UsedMemStatsTest) (string, error) {

	// testStat := UsedMemStats{
	// 	Pct:   opt.FloatWith(2.3),
	// 	Bytes: opt.UintWith(100),
	// }
	//zero := ZeroTest{Zero: false}
	// testStat := UsedMemStatsTest{

	// 	Raw:   1.0,
	// 	Iface: opt.FloatWith(2),
	// }
	wrapper := MarshalWrapper{
		Butterfly: &input,
	}

	val, err := json.MarshalIndent(&wrapper, " ", " ")
	if err != nil {
		return "", err
	}
	return string(val), nil
}
