package schema

type Errors []Error

func NewErrors() *Errors {
	return &Errors{}
}

func (errs *Errors) AddError(err *Error) {
	*errs = append(*errs, *err)
}

func (errs *Errors) AddErrors(errors *Errors) {
	if errors == nil {
		return
	}
	*errs = append(*errs, *errors...)
}

func (errs *Errors) HasRequiredErrors() bool {
	for _, err := range *errs {
		if err.IsType(RequiredType) {
			return true
		}
	}
	return false
}

func (errs *Errors) Error() string {
	error := "Required fields are missing: "
	for _, err := range *errs {
		if err.IsType(RequiredType) {
			error = error + "," + err.key
		}
	}
	return error
}

func (errs *Errors) ErrorDebug() string {
	error := "Fields are missing: "
	for _, err := range *errs {
		error = error + "," + err.key
	}
	return error
}
