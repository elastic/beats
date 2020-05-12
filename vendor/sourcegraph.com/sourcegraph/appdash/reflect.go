package appdash

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func flattenValue(prefix string, v reflect.Value, f func(k, v string)) {
	switch o := v.Interface().(type) {
	case time.Time:
		f(prefix, o.Format(time.RFC3339Nano))
		return
	case time.Duration:
		ms := float64(o.Nanoseconds()) / float64(time.Millisecond)
		f(prefix, strconv.FormatFloat(ms, 'f', -1, 64))
		return
	case fmt.Stringer:
		f(prefix, o.String())
		return
	}

	switch v.Kind() {
	case reflect.Ptr:
		flattenValue(prefix, v.Elem(), f)
	case reflect.Bool:
		f(prefix, strconv.FormatBool(v.Bool()))
	case reflect.Float32, reflect.Float64:
		f(prefix, strconv.FormatFloat(v.Float(), 'f', -1, 64))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f(prefix, strconv.FormatInt(v.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		f(prefix, strconv.FormatUint(v.Uint(), 10))
	case reflect.String:
		f(prefix, v.String())
	case reflect.Struct:
		for i, name := range fieldNames(v) {
			flattenValue(nest(prefix, name), v.Field(i), f)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			// small bit of cuteness here: use flattenValue on the key first,
			// then on the value
			flattenValue("", key, func(_, k string) {
				flattenValue(nest(prefix, k), v.MapIndex(key), f)
			})
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			flattenValue(nest(prefix, strconv.Itoa(i)), v.Index(i), f)
		}
	default:
		f(prefix, fmt.Sprintf("%+v", v.Interface()))
	}
}

func mapToKVs(m map[string]string) *[][2]string {
	var kvs [][2]string
	for k, v := range m {
		kvs = append(kvs, [2]string{k, v})
	}
	sort.Sort(kvsByKey(kvs))
	return &kvs
}

type kvsByKey [][2]string

func (v kvsByKey) Len() int           { return len(v) }
func (v kvsByKey) Less(i, j int) bool { return v[i][0] < v[j][0] }
func (v kvsByKey) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }

type structFieldsByName []reflect.StructField

func (v structFieldsByName) Len() int           { return len(v) }
func (v structFieldsByName) Less(i, j int) bool { return fieldName(v[i]) < fieldName(v[j]) }
func (v structFieldsByName) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }

func parseValue(as reflect.Type, s string) (reflect.Value, error) {
	vp, err := parseValueToPtr(as, s)
	if err != nil {
		return reflect.Value{}, err
	}
	if as.Kind() == reflect.Ptr {
		return vp, nil
	}
	return vp.Elem(), nil
}

func parseValueToPtr(as reflect.Type, s string) (reflect.Value, error) {
	switch [2]string{as.PkgPath(), as.Name()} {
	case [2]string{"time", "Time"}:
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(&t), nil
	case [2]string{"time", "Duration"}:
		// Multiply by 1000 because flattenValue divides by 1000.
		usec, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		d := time.Duration(usec * float64(time.Millisecond))
		return reflect.ValueOf(&d), nil
	}

	switch as.Kind() {
	case reflect.Ptr:
		return parseValueToPtr(as.Elem(), s)
	case reflect.Bool:
		vv, err := strconv.ParseBool(s)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(&vv), nil
	case reflect.Float32, reflect.Float64:
		vv, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		switch as.Kind() {
		case reflect.Float32:
			vvv := float32(vv)
			return reflect.ValueOf(&vvv), nil
		case reflect.Float64:
			return reflect.ValueOf(&vv), nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		vv, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		switch as.Kind() {
		case reflect.Int:
			vvv := int(vv)
			return reflect.ValueOf(&vvv), nil
		case reflect.Int8:
			vvv := int8(vv)
			return reflect.ValueOf(&vvv), nil
		case reflect.Int16:
			vvv := int16(vv)
			return reflect.ValueOf(&vvv), nil
		case reflect.Int32:
			vvv := int32(vv)
			return reflect.ValueOf(&vvv), nil
		case reflect.Int64:
			return reflect.ValueOf(&vv), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		vv, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		switch as.Kind() {
		case reflect.Uint:
			vvv := uint(vv)
			return reflect.ValueOf(&vvv), nil
		case reflect.Uint8:
			vvv := uint8(vv)
			return reflect.ValueOf(&vvv), nil
		case reflect.Uint16:
			vvv := uint16(vv)
			return reflect.ValueOf(&vvv), nil
		case reflect.Uint32:
			vvv := uint32(vv)
			return reflect.ValueOf(&vvv), nil
		case reflect.Uint64:
			return reflect.ValueOf(&vv), nil
		}
	case reflect.String:
		return reflect.ValueOf(&s), nil
	}
	return reflect.Value{}, nil
}

func unflattenValue(prefix string, v reflect.Value, t reflect.Type, kv *[][2]string) error {
	if !sort.IsSorted(kvsByKey(*kv)) {
		panic("unflattenValue: kv must be sorted (using kvsByKey)")
	}
	if len(*kv) == 0 {
		return nil
	}
	if !strings.HasPrefix((*kv)[0][0], prefix) && t.Kind() != reflect.Map { // map can have 0 fields
		*kv = (*kv)[1:]
		return unflattenValue(prefix, v, t, kv)
	}

	if v.IsValid() {
		treatAsValue := false
		switch v.Interface().(type) {
		case time.Time:
			treatAsValue = true
		}
		if treatAsValue {
			vv, err := parseValue(v.Type(), (*kv)[0][1])
			if err != nil {
				return err
			}
			if vv.Type().AssignableTo(v.Type()) {
				v.Set(vv)
			}
			*kv = (*kv)[1:]
			return nil
		}
	}

	switch t.Kind() {
	case reflect.Ptr:
		return unflattenValue(prefix, v, t.Elem(), kv)
	case reflect.Interface:
		return unflattenValue(prefix, v.Elem(), v.Type(), kv)
	case reflect.Bool, reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.String:
		vv, err := parseValue(v.Type(), (*kv)[0][1])
		if err != nil {
			return err
		}
		v.Set(vv)
		*kv = (*kv)[1:]
	case reflect.Struct:
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		var vtfs []reflect.StructField
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath == "" { // exported
				vtfs = append(vtfs, f)
			}
		}
		sort.Sort(structFieldsByName(vtfs))
		for _, vtf := range vtfs {
			vf := v.FieldByIndex(vtf.Index)
			if vf.IsValid() {
				fieldPrefix := nest(prefix, fieldName(vtf))
				if err := unflattenValue(fieldPrefix, vf, vtf.Type, kv); err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		m := reflect.MakeMap(t)
		keyPrefix := prefix + "."
		found := 0
		for _, kvv := range *kv {
			key, val := kvv[0], kvv[1]
			if key < keyPrefix {
				continue
			}
			if key > keyPrefix && !strings.HasPrefix(key, keyPrefix) {
				break
			}
			vv, err := parseValue(t.Elem(), val)
			if err != nil {
				return err
			}
			m.SetMapIndex(reflect.ValueOf(strings.TrimPrefix(key, keyPrefix)), vv)
			*kv = (*kv)[1:]
			found++
		}
		if found > 0 {
			v.Set(m)
		}
	case reflect.Slice, reflect.Array:
		keyPrefix := prefix + "."
		type elem struct {
			i int
			s string
		}
		var elems []elem
		maxI := 0
		for _, kvv := range *kv {
			key, val := kvv[0], kvv[1]
			if !strings.HasPrefix(key, keyPrefix) {
				break
			}

			i, err := strconv.Atoi(strings.TrimPrefix(key, keyPrefix))
			if err != nil {
				return err
			}

			elems = append(elems, elem{i, val})
			if i > maxI {
				maxI = i
			}

			*kv = (*kv)[1:]
		}

		if v.Kind() == reflect.Slice {
			v.Set(reflect.MakeSlice(t, maxI+1, maxI+1))
		}
		for _, e := range elems {
			vv, err := parseValue(t.Elem(), e.s)
			if err != nil {
				return err
			}
			v.Index(e.i).Set(vv)
		}
	}

	return nil
}

func fieldNames(v reflect.Value) map[int]string {
	t := v.Type()

	// check to see if a cached set exists
	cachedFieldNamesRW.RLock()
	m, ok := cachedFieldNames[t]
	cachedFieldNamesRW.RUnlock()

	if ok {
		return m
	}

	// otherwise, create it and return it
	cachedFieldNamesRW.Lock()
	m = make(map[int]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		fld := t.Field(i)
		if fld.PkgPath != "" {
			continue // ignore all unwant fields
		}

		m[i] = fieldName(fld)
	}
	cachedFieldNames[t] = m
	cachedFieldNamesRW.Unlock()
	return m
}

func fieldName(f reflect.StructField) string {
	name := f.Tag.Get("trace")
	if name == "" {
		name = f.Name
	}
	// TODO(sqs): check that the field name is only alphanumeric,
	// because we sort and assign special meaning to "."
	return name
}

var (
	cachedFieldNames   = make(map[reflect.Type]map[int]string, 20)
	cachedFieldNamesRW = new(sync.RWMutex)
)

func nest(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
