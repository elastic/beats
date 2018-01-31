package ucfg

func isCyclicError(err error) bool {
	switch v := err.(type) {
	case Error:
		return v.Reason() == ErrCyclicReference
	}
	return false
}

func isMissingError(err error) bool {
	switch v := err.(type) {
	case Error:
		return v.Reason() == ErrMissing
	}
	return false
}

func criticalResolveError(err error) bool {
	if err == nil {
		return false
	}
	return !(isCyclicError(err) || isMissingError(err))
}
