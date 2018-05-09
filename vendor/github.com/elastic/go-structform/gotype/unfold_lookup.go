package gotype

/*
func lookupReflUnfolder(t reflect.Type) (reflUnfolder, error) {
	// we always expect a pointer to a value
	bt := t.Elem()

	switch bt.Kind() {
	case reflect.Array:
		switch bt.Elem().Kind() {
		case reflect.Int:
			return liftGoUnfolder(newUnfolderArrInt()), nil
		}

	case reflect.Slice:

	case reflect.Map:
		if bt.Key().Kind() != reflect.String {
			return nil, errMapRequiresStringKey
		}

		switch bt.Elem().Kind() {
		case reflect.Interface:
			return liftGoUnfolder(newUnfolderMapIfc()), nil
		}

	case reflect.Struct:

	}

	return nil, errTODO()
}
*/
