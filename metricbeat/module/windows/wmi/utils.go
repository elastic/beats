package wmi

import (
	"fmt"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	wmi "github.com/microsoft/wmi/pkg/wmiinstance"
)

// Wrapper of the session.QueryInstances function that execute a query for at most a timeout
// Note that the underlying query will continue run
func ExecuteGuardedQueryInstances(session *wmi.WmiSession, query string, timeout time.Duration) ([]*wmi.WmiInstance, error) {
	var rows []*wmi.WmiInstance
	var err error
	done := make(chan bool)

	go func() {
		rows, err = session.QueryInstances(query)
		if err != nil {
			logp.Warn("Could not execute query %v", err)
		}
		done <- true
	}()

	select {
	case <-done:
		logp.Info("Query completed in time")
	case <-time.After(timeout):
		err = fmt.Errorf("query '%s' exceeded the timeout of %d", query, timeout)
		logp.Error(err)
	}

	return rows, err
}
