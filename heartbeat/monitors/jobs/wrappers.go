package jobs

import (
	"github.com/elastic/beats/heartbeat/eventext"
	"time"

	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// WrapCommon applies the common wrappers that all jobs get.
func WrapCommon(js []Job, id string, name string, typ string) []Job {
	return WrapAll(
		js,
		StatusWrapper,
		TimingWrapper,
		makeMetaWrapper(id, name, typ),
	)
}

func makeMetaWrapper(id string, name string, typ string) JobWrapper {
	return func(job Job) Job {
		return WithFields(
			common.MapStr{
				"monitor": common.MapStr{
					"id":   id,
					"name": name,
					"type": typ,
				},
			},
			job,
		)
	}
}

// StatusWrapper wraps the given Job's execution such that any error returned
// by the original Job will be set as a field. The original error will not be
// passed through as a return value. Errors may still be present but only if there
// is an actual error wrapping the error.
func StatusWrapper(job Job) Job {
	return AfterJob(job, func(event *beat.Event, cont []Job, err error) ([]Job, error) {
		fields := common.MapStr{
			"monitor": common.MapStr{
				"status": look.Status(err),
			},
		}
		if err != nil {
			fields["error"] = look.Reason(err)
		}
		eventext.MergeEventFields(event, fields)

		return cont, nil
	})
}

// TimingWrapper executes the given Job, checking the duration of its run and setting
// its status.
// It adds the monitor.duration and monitor.status fields.
func TimingWrapper(job Job) Job {
	return func(event *beat.Event) ([]Job, error) {
		start := time.Now()

		cont, err := job(event)

		if event != nil {
			eventext.MergeEventFields(event, common.MapStr{
				"monitor": common.MapStr{
					"duration": look.RTT(time.Since(start)),
				},
			})
			event.Timestamp = start
		}

		return cont, err
	}
}

// WithFields wraps a TaskRunner, updating all events returned with the set of
// fields configured.
func WithFields(fields common.MapStr, job Job) Job {
	return AfterJob(job, func(event *beat.Event, cont []Job, err error) ([]Job, error) {
		eventext.MergeEventFields(event, fields)

		return WrapAll(cont, func(job Job) Job {
			return WithFields(fields, job)
		}), err
	})
}
