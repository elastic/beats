package cleanup

// FailClean keeps track of functions to be executed of FailClean did
// not receive a success signal.
type FailClean struct {
	success bool
	fns     []func()
}

// Signal sends a success or fail signal to FailClean.
func (f *FailClean) Signal(success bool) {
	f.success = success
}

// Add adds another cleanup handler. The last added handler will be run first.
func (f *FailClean) Add(fn func()) {
	f.fns = append(f.fns, fn)
}

// Cleanup runs all cleanup handlers in reverse order.
func (f *FailClean) Cleanup() {
	if f.success {
		return
	}

	for i := len(f.fns) - 1; i >= 0; i-- {
		f.fns[i]()
	}
}
