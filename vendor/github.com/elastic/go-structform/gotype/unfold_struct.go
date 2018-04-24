package gotype

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"

	structform "github.com/elastic/go-structform"
)

type unfolderStruct struct {
	unfolderErrExpectKey
	fields map[string]fieldUnfolder
}

type unfolderStructStart struct {
	unfolderErrObjectStart
}

type fieldUnfolder struct {
	offset    uintptr
	initState func(ctx *unfoldCtx, sp unsafe.Pointer)
}

var (
	_singletonUnfolderStructStart = &unfolderStructStart{}

	_ignoredField = &fieldUnfolder{
		initState: _singletonUnfoldIgnorePtr.initState,
	}
)

func createUnfolderReflStruct(ctx *unfoldCtx, t reflect.Type) (*unfolderStruct, error) {
	// assume t is pointer to struct
	t = t.Elem()

	fields, err := fieldUnfolders(ctx, t)
	if err != nil {
		return nil, err
	}

	u := &unfolderStruct{fields: fields}
	return u, nil
}

func fieldUnfolders(ctx *unfoldCtx, t reflect.Type) (map[string]fieldUnfolder, error) {
	count := t.NumField()
	fields := map[string]fieldUnfolder{}

	for i := 0; i < count; i++ {
		st := t.Field(i)

		name := st.Name
		rune, _ := utf8.DecodeRuneInString(name)
		if !unicode.IsUpper(rune) {
			continue
		}

		tagName, tagOpts := parseTags(st.Tag.Get(ctx.opts.tag))
		if tagOpts.squash {
			if st.Type.Kind() != reflect.Struct {
				return nil, errSquashNeedObject
			}

			sub, err := fieldUnfolders(ctx, st.Type)
			if err != nil {
				return nil, err
			}

			for name, fu := range sub {
				fu.offset += st.Offset
				if _, exists := fields[name]; exists {
					return nil, fmt.Errorf("duplicate field name %v", name)
				}

				fields[name] = fu
			}
		} else {
			if tagName != "" {
				name = tagName
			} else {
				name = strings.ToLower(name)
			}

			if _, exists := fields[name]; exists {
				return nil, fmt.Errorf("duplicate field name %v", name)
			}

			fu, err := makeFieldUnfolder(ctx, st)
			if err != nil {
				return nil, err
			}

			fields[name] = fu
		}
	}

	return fields, nil
}

func makeFieldUnfolder(ctx *unfoldCtx, st reflect.StructField) (fieldUnfolder, error) {
	fu := fieldUnfolder{offset: st.Offset}

	if pu := lookupGoPtrUnfolder(st.Type); pu != nil {
		fu.initState = pu.initState
	} else {
		targetType := reflect.PtrTo(st.Type)
		ru, err := lookupReflUnfolder(ctx, targetType)
		if err != nil {
			return fu, err
		}

		if su, ok := ru.(*unfolderStruct); ok {
			fu.initState = su.initStatePtr
		} else {
			fu.initState = wrapReflUnfolder(st.Type, ru)
		}
	}

	return fu, nil
}

func wrapReflUnfolder(t reflect.Type, ru reflUnfolder) func(*unfoldCtx, unsafe.Pointer) {
	return func(ctx *unfoldCtx, ptr unsafe.Pointer) {
		v := reflect.NewAt(t, ptr)
		ru.initState(ctx, v)
	}
}

func (u *unfolderStruct) initState(ctx *unfoldCtx, v reflect.Value) {
	u.initStatePtr(ctx, unsafe.Pointer(v.Pointer()))
}

func (u *unfolderStruct) initStatePtr(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.ptr.push(ptr)
	ctx.unfolder.push(u)
	ctx.unfolder.push(_singletonUnfolderStructStart)
}

func (u *unfolderStructStart) OnObjectStart(ctx *unfoldCtx, l int, bt structform.BaseType) error {
	ctx.unfolder.pop()
	return nil
}

func (u *unfolderStruct) OnObjectFinished(ctx *unfoldCtx) error {
	ctx.unfolder.pop()
	ctx.ptr.pop()
	return nil
}

func (u *unfolderStruct) OnChildObjectDone(ctx *unfoldCtx) error { return nil }
func (u *unfolderStruct) OnChildArrayDone(ctx *unfoldCtx) error  { return nil }

func (u *unfolderStruct) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, bytes2Str(key))
}

func (u *unfolderStruct) OnKey(ctx *unfoldCtx, key string) error {
	field, exists := u.fields[key]
	if !exists {
		_ignoredField.initState(ctx, nil)
		return nil
	}

	structPtr := ctx.ptr.current
	fieldPtr := unsafe.Pointer(uintptr(structPtr) + field.offset)
	field.initState(ctx, fieldPtr)
	return nil
}
