package gotype

import "reflect"

type initOptions struct {
	foldFns map[reflect.Type]reFoldFn
}

type Option func(*initOptions) error

func applyOpts(opts []Option) (initOptions, error) {
	i := initOptions{}
	for _, o := range opts {
		if err := o(&i); err != nil {
			return initOptions{}, err
		}
	}
	return i, nil
}

func Folders(in ...interface{}) Option {
	folders, err := makeUserFoldFns(in)
	if err != nil {
		return func(_ *initOptions) error { return err }
	}

	if len(folders) == 0 {
		return func(*initOptions) error { return nil }
	}

	return func(o *initOptions) error {
		if o.foldFns == nil {
			o.foldFns = map[reflect.Type]reFoldFn{}
		}

		for k, v := range folders {
			o.foldFns[k] = v
		}
		return nil
	}
}

func makeUserFoldFns(in []interface{}) (map[reflect.Type]reFoldFn, error) {
	M := map[reflect.Type]reFoldFn{}

	for _, v := range in {
		fn := reflect.ValueOf(v)
		fptr, err := makeUserFoldFn(fn)
		if err != nil {
			return nil, err
		}

		ta0 := fn.Type().In(0)
		M[ta0] = liftUserPtrFn(fptr)
		M[ta0.Elem()] = liftUserValueFn(fptr)
	}

	return M, nil
}
