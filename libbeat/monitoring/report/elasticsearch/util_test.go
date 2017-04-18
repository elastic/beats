package elasticsearch

import (
	"testing"
	"time"
)

func TestStopper(t *testing.T) {
	runPar := func(name string, f func(*testing.T)) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f(t)
		})
	}

	st := newStopper()
	runPar("wait on channel stop", func(*testing.T) { <-st.C() })
	runPar("use wait", func(*testing.T) { st.Wait() })
	runPar("use dowait", func(t *testing.T) {
		i := 0
		st.DoWait(func() { i = 1 })
		if i != 1 {
			t.Error("callback did not run")
		}
	})

	// unblock all waiters
	time.Sleep(10 * time.Millisecond)
	st.Stop()

	// test either blocks or returns as stopper as been stopped
	t.Run("wait after stop", func(t *testing.T) { st.Wait() })

	// check subsequent stop does not panic
	st.Stop()
	st.Stop()
}
