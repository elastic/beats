package scheduling

type (
	SigDropEvent struct{}

	SigCloseClient struct{}
)

var SigDrop = &SigDropEvent{}
var SigClose = &SigCloseClient{}

func (sig *SigDropEvent) IsClose() bool { return false }
func (sig *SigDropEvent) IsDrop() bool  { return true }
func (sig *SigDropEvent) Error() string { return "signal: event drop" }

func (sig *SigCloseClient) IsClose() bool { return true }
func (sig *SigCloseClient) IsDrop() bool  { return true }
func (sig *SigCloseClient) Error() string { return "signal: closing" }

func IsDropSignal(err error) bool {
	type ifc interface {
		IsDrop() bool
	}

	if sig, ok := err.(ifc); ok {
		return sig.IsDrop()
	}

	return false
}

func IsCloseSignal(err error) bool {
	type ifc interface {
		IsClose() bool
	}

	if sig, ok := err.(ifc); ok {
		return sig.IsClose()
	}

	return false
}
