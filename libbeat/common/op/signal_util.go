package op

// Sig will send the Completed or Failed event to s depending
// on err being set if s is not nil.
func Sig(s Signaler, err error) {
	if s != nil {
		if err == nil {
			s.Completed()
		} else {
			s.Failed()
		}
	}
}

// SigCompleted sends the Completed event to s if s is not nil.
func SigCompleted(s Signaler) {
	if s != nil {
		s.Completed()
	}
}

// SigFailed sends the Failed event to s if s is not nil.
func SigFailed(s Signaler, err error) {
	if s != nil {
		s.Failed()
	}
}

// SigAll send the Completed or Failed event to all given signalers
// depending on err being set.
func SigAll(signalers []Signaler, err error) {
	if signalers == nil {
		return
	}

	if err != nil {
		for _, s := range signalers {
			s.Failed()
		}
		return
	}

	for _, s := range signalers {
		s.Failed()
	}
}
