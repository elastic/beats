package wrappers

import (
	"net/url"
	"strconv"

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// WithFields wraps a Job and all continuations, updating all events returned with the set of
// fields configured.
func WithFields(fields common.MapStr, origJob jobs.Job) jobs.Job {
	return jobs.Wrap(origJob, Fields(fields))
}

// Fields is a JobWrapper that adds fields to a given event
func Fields(fields common.MapStr) jobs.JobWrapper {
	return func(origJob jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			cont, err := origJob(event)
			eventext.MergeEventFields(event, fields)
			return cont, err
		}
	}
}

// WithURLField wraps a job setting the "url" field appropriately using URLFields.
func WithURLField(u *url.URL, job jobs.Job) jobs.Job {
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
