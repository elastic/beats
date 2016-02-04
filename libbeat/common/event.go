package common

import (
	"fmt"
	"reflect"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/mitchellh/reflectwalk"
)

type EventWalker struct {
}

func (w EventWalker) Struct(v reflect.Value) error {
	if v.Type().String() == "common.Time" {
		return nil
	}
	if v.Type().String() == "time.Location" {
		return nil
	}
	return fmt.Errorf("no struct allowed: %s, type=%v", v.String(), v.Type().String())
}

func (w EventWalker) StructField(v reflect.StructField, f reflect.Value) error {
	return nil
}

func (w EventWalker) Primitive(v reflect.Value) error {
	return nil
}

func (w EventWalker) Map(v reflect.Value) error {
	return nil
}

func CheckEvent(event MapStr) error {
	logp.Info("checking event")
	var walker EventWalker

	if err := reflectwalk.Walk(event, walker); err != nil {
		return err
	}
	return nil
}
