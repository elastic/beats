package sftest

import structform "github.com/urso/go-structform"

func Arr(l int, t structform.BaseType, elems ...interface{}) []Record {
	a := []Record{ArrayStartRec{l, t}}
	for _, elem := range elems {
		switch v := elem.(type) {
		case Record:
			a = append(a, v)
		case []Record:
			a = append(a, v...)
		case Recording:
			a = append(a, v...)
		default:
			panic("invalid key type")
		}
	}

	return append(a, ArrayFinishRec{})
}

func Obj(l int, t structform.BaseType, kv ...interface{}) []Record {
	if len(kv)%2 != 0 {
		panic("invalid object")
	}

	a := []Record{ObjectStartRec{l, t}}
	for i := 0; i < len(kv); i += 2 {
		k := kv[i].(string)
		a = append(a, ObjectKeyRec{k})

		switch v := kv[i+1].(type) {
		case Record:
			a = append(a, v)
		case []Record:
			a = append(a, v...)
		case Recording:
			a = append(a, v...)
		default:
			panic("invalid key type")
		}
	}

	return append(a, ObjectFinishRec{})
}
