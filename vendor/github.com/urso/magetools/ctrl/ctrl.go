package ctrl

import "strings"

type Operation func() error

// ForEachFrom creates and runs an operation for every element in the list returned by listGen.
// Typical modes are Sequential (fail fast), or Each (collect errors).
func ForEachFrom(
	listGen func() ([]string, error),
	mode func(...Operation) error,
	operation func(string) error,
) error {
	list, err := listGen()
	if err != nil {
		return err
	}

	return ForEach(list, mode, operation)
}

// ForEach creates and runs an operation for every element in the list.
// Typical modes are Sequential (fail fast), or Each (collect errors).
func ForEach(
	list []string,
	mode func(...Operation) error,
	operation func(string) error,
) error {
	ops := make([]Operation, len(list))
	for i, v := range list {
		v := v
		ops[i] = func() error { return operation(v) }
	}
	return mode(ops...)
}

// Sequential runs a list of operations. It returns on the first failure
func Sequential(ops ...Operation) error {
	for _, op := range ops {
		if err := op(); err != nil {
			return err
		}
	}
	return nil
}

// Each runs each single operation, collecting errors.
func Each(ops ...Operation) error {
	var errs []error
	for _, op := range ops {
		if err := op(); err != nil {
			errs = append(errs, err)
		}
	}
	return makeErrs(errs)
}

type multiErr []error

func makeErrs(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return multiErr(errs)
	}
}

func (m multiErr) Error() string {
	var bld strings.Builder
	for _, err := range m {
		if bld.Len() > 0 {
			bld.WriteByte('\n')
			bld.WriteString(err.Error())
		}
	}
	return bld.String()
}
