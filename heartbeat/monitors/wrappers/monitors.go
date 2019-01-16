package wrappers

import (
	"time"

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// WrapCommon applies the common wrappers that all monitor jobs get.
func WrapCommon(js []jobs.Job, id string, name string, typ string) []jobs.Job {
	return jobs.WrapAll(
		js,
		monStatus,
		monTiming,
		monMeta(id, name, typ),
	)
}

func monMeta(id string, name string, typ string) jobs.JobWrapper {
	return func(job jobs.Job) jobs.Job {
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

// monStatus wraps the given Job's execution such that any error returned
// by the original Job will be set as a field. The original error will not be
// passed through as a return value. Errors may still be present but only if there
// is an actual error wrapping the error.
func monStatus(origJob jobs.Job) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		cont, err := origJob(event)
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
	}
}

// monTiming executes the given Job, checking the duration of its run and setting
// its monStatus.
// It adds the monitor.duration and monitor.monStatus fields.
func monTiming(job jobs.Job) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
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
