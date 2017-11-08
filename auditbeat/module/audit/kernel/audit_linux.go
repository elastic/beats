package kernel

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/go-libaudit"
	"github.com/elastic/go-libaudit/aucoalesce"
	"github.com/elastic/go-libaudit/auparse"
)

const (
	metricsetName = "audit.kernel"
	logPrefix     = "[" + metricsetName + "]"
)

var (
	debugf = logp.MakeDebug(metricsetName)

	auditMetrics = monitoring.Default.NewRegistry(metricsetName)
	lostMetric   = monitoring.NewInt(auditMetrics, "lost")
)

func init() {
	if err := mb.Registry.AddMetricSet("audit", "kernel", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
}

// MetricSet listens for audit messages from the Linux kernel using a netlink
// socket. It buffers the messages to ensure ordering and then streams the
// output. MetricSet implements the mb.PushMetricSet interface, and therefore
// does not rely on polling.
type MetricSet struct {
	mb.BaseMetricSet
	config Config
	client *libaudit.AuditClient
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The %v metricset is a beta feature", metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrap(err, "failed to unpack the audit.kernel config")
	}

	_, _, kernel, _ := kernelVersion()
	debugf("the metricset is running as euid=%v on kernel=%v", os.Geteuid(), kernel)

	client, err := newAuditClient(&config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create audit.kernel client")
	}

	lostMetric.Set(0)

	return &MetricSet{
		BaseMetricSet: base,
		client:        client,
		config:        config,
	}, nil
}

func newAuditClient(c *Config) (*libaudit.AuditClient, error) {
	hasMulticast := hasMulticastSupport()

	switch c.SocketType {
	// Attempt to determine the optimal socket_type.
	case "":
		// Use multicast only when no rules are present. Specifying rules
		// implies you want control over the audit framework so you should be
		// using unicast.
		if rules, _ := c.rules(); len(rules) == 0 && hasMulticast {
			c.SocketType = "multicast"
			logp.Info("%v kernel.socket_type=multicast will be used.", logPrefix)
		}
	case "multicast":
		if !hasMulticast {
			logp.Warn("%v kernel.socket_type is set to multicast "+
				"but based on the kernel version multicast audit subscriptions "+
				"are not supported. unicast will be used instead.", logPrefix)
			c.SocketType = "unicast"
		}
	}

	switch c.SocketType {
	case "multicast":
		return libaudit.NewMulticastAuditClient(nil)
	default:
		c.SocketType = "unicast"
		return libaudit.NewAuditClient(nil)
	}
}

// Run initializes the audit client and receives audit messages from the
// kernel until the reporter's done channel is closed.
func (ms *MetricSet) Run(reporter mb.PushReporter) {
	defer ms.client.Close()

	if err := ms.addRules(reporter); err != nil {
		reporter.Error(err)
		logp.Err("%v %v", logPrefix, err)
		return
	}

	out, err := ms.receiveEvents(reporter.Done())
	if err != nil {
		reporter.Error(err)
		logp.Err("%v %v", logPrefix, err)
		return
	}

	for {
		select {
		case <-reporter.Done():
			return
		case msgs := <-out:
			event, err := buildMapStr(msgs, ms.config)
			if err != nil {
				reporter.ErrorWith(err, event)
			} else {
				reporter.Event(event)
			}
		}
	}
}

func (ms *MetricSet) addRules(reporter mb.PushReporter) error {
	rules, err := ms.config.rules()
	if err != nil {
		return errors.Wrap(err, "failed to add rules")
	}

	if len(rules) == 0 {
		logp.Info("%v No audit kernel.rules were specified.", logPrefix)
		return nil
	}

	client, err := libaudit.NewAuditClient(nil)
	if err != nil {
		return errors.Wrap(err, "failed to create audit client for adding rules")
	}
	defer client.Close()

	// Delete existing rules.
	n, err := client.DeleteRules()
	if err != nil {
		return errors.Wrap(err, "failed to delete existing rules")
	}
	logp.Info("%v Deleted %v pre-existing audit rules.", logPrefix, n)

	// Add rules from config.
	var failCount int
	for _, rule := range rules {
		if err = client.AddRule(rule.data); err != nil {
			// Treat rule add errors as warnings and continue.
			err = errors.Wrapf(err, "failed to add kernel rule '%v'", rule.flags)
			reporter.Error(err)
			logp.Warn("%v %v", logPrefix, err)
			failCount++
		}
	}
	logp.Info("%v Successfully added %d of %d kernel audit rules.",
		logPrefix, len(rules)-failCount, len(rules))
	return nil
}

func (ms *MetricSet) initClient() error {
	if ms.config.SocketType == "multicast" {
		// This request will fail with EPERM if this process does not have
		// CAP_AUDIT_CONTROL, but we will ignore the response. The user will be
		// required to ensure that auditing is enabled if the process is only
		// given CAP_AUDIT_READ.
		err := ms.client.SetEnabled(true, libaudit.NoWait)
		return errors.Wrap(err, "failed to enable auditing in the kernel")
	}

	// Unicast client initialization (requires CAP_AUDIT_CONTROL and that the
	// process be in initial PID namespace).
	status, err := ms.client.GetStatus()
	if err != nil {
		return errors.Wrap(err, "failed to get audit status")
	}
	debugf("audit status from kernel at start: status=%+v", status)

	if fm, _ := ms.config.failureMode(); status.Failure != fm {
		if err = ms.client.SetFailure(libaudit.FailureMode(fm), libaudit.NoWait); err != nil {
			return errors.Wrap(err, "failed to set audit failure mode in kernel")
		}
	}

	if status.RateLimit != ms.config.RateLimit {
		if err = ms.client.SetRateLimit(ms.config.RateLimit, libaudit.NoWait); err != nil {
			return errors.Wrap(err, "failed to set audit rate limit in kernel")
		}
	}

	if status.BacklogLimit != ms.config.BacklogLimit {
		if err = ms.client.SetBacklogLimit(ms.config.BacklogLimit, libaudit.NoWait); err != nil {
			return errors.Wrap(err, "failed to set audit backlog limit in kernel")
		}
	}

	if status.Enabled == 0 {
		if err = ms.client.SetEnabled(true, libaudit.NoWait); err != nil {
			return errors.Wrap(err, "failed to enable auditing in the kernel")
		}
	}

	if err := ms.client.SetPID(libaudit.NoWait); err != nil {
		return errors.Wrap(err, "failed to set audit PID")
	}

	return nil
}

func (ms *MetricSet) receiveEvents(done <-chan struct{}) (<-chan []*auparse.AuditMessage, error) {
	if err := ms.initClient(); err != nil {
		return nil, err
	}

	out := make(chan []*auparse.AuditMessage, ms.config.StreamBufferQueueSize)
	reassembler, err := libaudit.NewReassembler(int(ms.config.ReassemblerMaxInFlight), ms.config.ReassemblerTimeout, &stream{done, out})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Reassembler")
	}
	go maintain(done, reassembler)

	go func() {
		defer close(out)
		defer reassembler.Close()

		for {
			raw, err := ms.client.Receive(false)
			if err != nil {
				continue
			}

			if filterRecordType(raw.Type) {
				continue
			}

			if err := reassembler.Push(raw.Type, raw.Data); err != nil {
				debugf("dropping message record_type=%v message='%v': ",
					raw.Type, string(raw.Data), err)
				continue
			}
		}
	}()

	return out, nil
}

// maintain periodically evicts timed-out events from the Reassembler. This
// function will block until the done channel is closed or the Reassembler is
// closed.
func maintain(done <-chan struct{}, reassembler *libaudit.Reassembler) {
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-done:
			return
		case <-tick.C:
			if err := reassembler.Maintain(); err != nil {
				return
			}
		}
	}
}

func filterRecordType(typ auparse.AuditMessageType) bool {
	// Messages from 1300-2999 are valid audit message types.
	if typ < auparse.AUDIT_USER_AUTH || typ > auparse.AUDIT_LAST_USER_MSG2 {
		return true
	}

	return false
}

func buildMapStr(msgs []*auparse.AuditMessage, config Config) (common.MapStr, error) {
	event, err := aucoalesce.CoalesceMessages(msgs)
	if err != nil {
		// Add messages on error so that it's possible to debug the problem.
		m := common.MapStr{}
		addMessages(msgs, m)
		return m, err
	}

	if config.ResolveIDs {
		aucoalesce.ResolveIDs(event)
	}

	m := common.MapStr{
		"@timestamp":  event.Timestamp,
		"sequence":    event.Sequence,
		"category":    event.Category.String(),
		"record_type": strings.ToLower(event.Type.String()),
		"result":      event.Result,
		"session":     event.Session,
		"data":        event.Data,
	}
	if event.Subject.Primary != "" {
		m.Put("actor.primary", event.Subject.Primary)
	}
	if event.Subject.Secondary != "" {
		m.Put("actor.secondary", event.Subject.Secondary)
	}
	if len(event.Subject.Attributes) > 0 {
		m.Put("actor.attrs", event.Subject.Attributes)
	}
	if len(event.Subject.SELinux) > 0 {
		m.Put("actor.selinux", event.Subject.SELinux)
	}
	if event.Object.Primary != "" {
		m.Put("thing.primary", event.Object.Primary)
	}
	if event.Object.Secondary != "" {
		m.Put("thing.secondary", event.Object.Secondary)
	}
	if event.Object.What != "" {
		m.Put("thing.what", event.Object.What)
	}
	if len(event.Object.SELinux) > 0 {
		m.Put("thing.selinux", event.Object.SELinux)
	}
	if event.Action != "" {
		m.Put("action", event.Action)
	}
	if event.How != "" {
		m.Put("how", event.How)
	}
	if event.Key != "" {
		m.Put("key", event.Key)
	}
	if len(event.Paths) > 0 {
		m.Put("paths", event.Paths)
	}
	if len(event.Socket) > 0 {
		m.Put("socket", event.Socket)
	}
	if config.Warnings && len(event.Warnings) > 0 {
		warnings := make([]string, 0, len(event.Warnings))
		for _, err := range event.Warnings {
			warnings = append(warnings, err.Error())
		}
		m.Put("warnings", warnings)
		addMessages(msgs, m)
	}
	if config.RawMessage {
		addMessages(msgs, m)
	}

	return m, nil
}

func addMessages(msgs []*auparse.AuditMessage, m common.MapStr) {
	_, added := m["messages"]
	if !added && len(msgs) > 0 {
		rawMsgs := make([]string, 0, len(msgs))
		for _, msg := range msgs {
			rawMsgs = append(rawMsgs, "type="+msg.RecordType.String()+" msg="+msg.RawData)
		}
		m["messages"] = rawMsgs
	}
}

// stream type

// stream receives callbacks from the libaudit.Reassmbler for completed events
// or lost events that are detected by gaps in sequence numbers.
type stream struct {
	done <-chan struct{}
	out  chan<- []*auparse.AuditMessage
}

func (s *stream) ReassemblyComplete(msgs []*auparse.AuditMessage) {
	select {
	case <-s.done:
		return
	case s.out <- msgs:
	}
}

func (s *stream) EventsLost(count int) {
	lostMetric.Inc()
}

func hasMulticastSupport() bool {
	// Check the kernel version because 3.16+ should have multicast
	// support.
	major, minor, _, err := kernelVersion()
	if err != nil {
		// Assume not supported.
		return false
	}

	switch {
	case major > 3,
		major == 3 && minor >= 16:
		return true
	}

	return false
}

func kernelVersion() (major, minor int, full string, err error) {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		return 0, 0, "", err
	}

	data := make([]byte, len(uname.Release))
	for i, v := range uname.Release {
		if v == 0 {
			break
		}
		data[i] = byte(v)
	}

	release := string(data)
	parts := strings.SplitN(release, ".", 3)
	if len(parts) < 2 {
		return 0, 0, release, errors.Errorf("failed to parse uname release '%v'", release)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, release, errors.Wrapf(err, "failed to parse major version from '%v'", release)
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, release, errors.Wrapf(err, "failed to parse minor version from '%v'", release)
	}

	return major, minor, release, nil
}
