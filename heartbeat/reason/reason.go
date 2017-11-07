package reason

import "github.com/elastic/beats/libbeat/common"

type Reason interface {
	error
	Type() string
}

type ValidateError struct {
	err error
}

type IOError struct {
	err error
}

func ValidateFailed(err error) Reason {
	if err == nil {
		return nil
	}
	return ValidateError{err}
}

func IOFailed(err error) Reason {
	if err == nil {
		return nil
	}
	return IOError{err}
}

func (e ValidateError) Error() string { return e.err.Error() }
func (ValidateError) Type() string    { return "validate" }

func (e IOError) Error() string { return e.err.Error() }
func (IOError) Type() string    { return "io" }

func FailError(typ string, err error) common.MapStr {
	return common.MapStr{
		"type":    typ,
		"message": err.Error(),
	}
}

func Fail(r Reason) common.MapStr {
	return common.MapStr{
		"type":    r.Type(),
		"message": r.Error(),
	}
}

func FailIO(err error) common.MapStr { return Fail(IOError{err}) }

func FailValidate(err error) common.MapStr { return Fail(ValidateError{err}) }
