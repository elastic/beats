package jobs

import "github.com/elastic/beats/libbeat/beat"

// A Job represents a unit of execution, and may return multiple continuation jobs.
type Job func(event *beat.Event) ([]Job, error)

// MakeSimpleJob creates a new Job from a callback function. The callback should
// return an valid event and can not create any sub-tasks to be executed after
// completion.
func MakeSimpleJob(f func(*beat.Event) error) Job {
	return func(event *beat.Event) ([]Job, error) {
		return nil, f(event)
	}
}

// JobWrapper is used for functions that wrap other jobs transforming their behavior.
type JobWrapper func(Job) Job

// WrapAll wraps all jobs and their continuations with the given wrappers
func WrapAll(jobs []Job, wrappers ...JobWrapper) []Job {
	var wrapped []Job
	for _, j := range jobs {
		for _, wrapper := range wrappers {
			j = Wrap(j, wrapper)
		}
		wrapped = append(wrapped, j)
	}
	return wrapped
}

// Wrap wraps the given Job and also any continuations with the given JobWrapper.
func Wrap(job Job, wrapper JobWrapper) Job {
	return func(event *beat.Event) ([]Job, error) {
		cont, err := wrapper(job)(event)
		return WrapAll(cont, wrapper), err
	}
}
