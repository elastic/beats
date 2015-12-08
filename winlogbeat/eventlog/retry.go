package eventlog

// retry invokes the retriable function. If the retriable function returns an
// error then the corrective action function is invoked and passed the error.
// The correctiveAction function should attempt to correct the error so that
// retriable can be invoked again.
func retry(retriable func() error, correctiveAction func(error) error) error {
	err := retriable()
	if err != nil {
		caErr := correctiveAction(err)
		if caErr != nil {
			// Something went wrong, return original error.
			return err
		}

		retryErr := retriable()
		if retryErr != nil {
			// The second attempt failed, return original error.
			return err
		}
	}

	return nil
}
