package jobs

import (
	"github.com/elastic/beats/heartbeat/eventext"
	"net/url"
	"strconv"
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

// WithURLField wraps a job setting the "url" field appropriately using URLFields.
func WithURLField(u *url.URL, job Job) Job {
	return WithFields(common.MapStr{"url": URLFields(u)}, job)
}

// URLFields generates ECS compatible URL.* fields from a given url. It also sanitizes
// the password making sure that, if present, it is replaced with the string '<hidden>'.
func URLFields(u *url.URL) common.MapStr {
	fields := common.MapStr{
		"scheme": u.Scheme,
		"domain": u.Hostname(),
	}

	if u.Port() != "" {
		fields["port"], _ = strconv.ParseUint(u.Port(), 10, 8)
	}

	if u.Path != "" {
		fields["path"] = u.Path
	}

	if u.RawQuery != "" {
		fields["query"] = u.RawQuery
	}

	if u.User != nil {
		if u.User.Username() != "" {
			fields["username"] = u.User.Username()
		}
		if _, ok := u.User.Password(); ok {
			// Sanitize the password if present
			hiddenPass := "<hidden>"
			u.User = url.UserPassword(u.User.Username(), hiddenPass)
			fields["password"] = hiddenPass
		}
	}

	// This is called last to ensure that the password is sanitized
	fields["full"] = u.String()

	return fields
}
