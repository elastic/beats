package common

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

func MarshallUnmarshall(v interface{}) (MapStr, error) {
	// decode and encode JSON
	marshaled, err := json.Marshal(v)
	if err != nil {
		logp.Warn("marshal err: %v", err)
		return nil, err
	}
	var v1 MapStr
	err = json.Unmarshal(marshaled, &v1)
	if err != nil {
		logp.Warn("unmarshal err: %v")
		return nil, err
	}

	return v1, nil
}

func ConvertToGenericEvent(v MapStr) MapStr {

	for key, value := range v {

		switch value.(type) {
		case Time, *Time:
			continue
		case time.Location, *time.Location:
			continue
		case MapStr:
			v[key] = ConvertToGenericEvent(value.(MapStr))
			continue
		case *MapStr:
			v[key] = ConvertToGenericEvent(*value.(*MapStr))
			continue
		default:

			typ := reflect.TypeOf(value)

			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}

			switch typ.Kind() {
			case reflect.Bool:
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			case reflect.Uintptr:
			case reflect.Float32, reflect.Float64:
			case reflect.Complex64, reflect.Complex128:
			case reflect.String:
			case reflect.UnsafePointer:
			case reflect.Array, reflect.Slice:
			//case reflect.Chan:
			//case reflect.Func:
			//case reflect.Interface:
			case reflect.Map:
				anothermap, err := MarshallUnmarshall(value)
				if err != nil {
					logp.Warn("fail to marschall & unmarshall map (%v): key=%v value=%#v",
						key, value)
					continue
				}
				v[key] = anothermap

			case reflect.Struct:
				anothermap, err := MarshallUnmarshall(value)
				if err != nil {
					logp.Warn("fail to marschall & unmarshall struct %v", key)
					continue
				}
				v[key] = anothermap
			default:
				logp.Warn("unknown type %v", typ)
				continue
			}
		}
	}
	return v
}
