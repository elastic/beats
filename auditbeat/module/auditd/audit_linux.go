package auditd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/go-libaudit"
	"github.com/elastic/go-libaudit/aucoalesce"
	"github.com/elastic/go-libaudit/auparse"
)

const (
	namespace = "auditd"

	auditLocked = 2

	unicast   = "unicast"
	multicast = "multicast"
)

var (
	auditdMetrics = monitoring.Default.NewRegistry(moduleName)
	lostMetric    = monitoring.NewInt(auditdMetrics, "lost")
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
		mb.WithHostParser(parse.EmptyHostParser),
		mb.WithNamespace(namespace),
	)
}

// MetricSet listens for audit messages from the Linux kernel using a netlink
// socket. It buffers the messages to ensure ordering and then streams the
// output. MetricSet implements the mb.PushMetricSet interface, and therefore
// does not rely on polling.
type MetricSet struct {
	mb.BaseMetricSet
	config Config
	client *libaudit.AuditClient
	log    *logp.Logger
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrap(err, "failed to unpack the auditd config")
	}

	log := logp.NewLogger(moduleName)
	_, _, kernel, _ := kernelVersion()
	log.Infof("auditd module is running as euid=%v on kernel=%v", os.Geteuid(), kernel)

	client, err := newAuditClient(&config, log)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create audit client")
	}

	lostMetric.Set(0)

	return &MetricSet{
		BaseMetricSet: base,
		client:        client,
		config:        config,
		log:           log,
	}, nil
}

func newAuditClient(c *Config, log *logp.Logger) (*libaudit.AuditClient, error) {
	var err error
	c.SocketType, err = determineSocketType(c, log)
	if err != nil {
		return nil, err
	}
	log.Infof("socket_type=%s will be used.", c.SocketType)

	if c.SocketType == multicast {
		return libaudit.NewMulticastAuditClient(nil)
	}
	return libaudit.NewAuditClient(nil)
}

// Run initializes the audit client and receives audit messages from the
// kernel until the reporter's done channel is closed.
func (ms *MetricSet) Run(reporter mb.PushReporterV2) {
	defer ms.client.Close()

	if err := ms.addRules(reporter); err != nil {
		reporter.Error(err)
		ms.log.Errorw("Failure adding audit rules", "error", err)
		return
	}

	out, err := ms.receiveEvents(reporter.Done())
	if err != nil {
		reporter.Error(err)
		ms.log.Errorw("Failure receiving audit events", "error", err)
		return
	}

	for {
		select {
		case <-reporter.Done():
			return
		case msgs := <-out:
			reporter.Event(buildMetricbeatEvent(msgs, ms.config))
		}
	}
}

func (ms *MetricSet) addRules(reporter mb.PushReporterV2) error {
	rules, err := ms.config.rules()
	if err != nil {
		return errors.Wrap(err, "failed to add rules")
	}

	if len(rules) == 0 {
		ms.log.Info("No audit_rules were specified.")
		return nil
	}

	client, err := libaudit.NewAuditClient(nil)
	if err != nil {
		return errors.Wrap(err, "failed to create audit client for adding rules")
	}
	defer client.Close()

	// Don't attempt to change configuration if audit rules are locked (enabled == 2).
	// Will result in EPERM.
	status, err := client.GetStatus()
	if err != nil {
		err = errors.Wrap(err, "failed to get audit status before adding rules")
		reporter.Error(err)
		return err
	}
	if status.Enabled == auditLocked {
		return errors.New("Skipping rule configuration: Audit rules are locked")
	}

	// Delete existing rules.
	n, err := client.DeleteRules()
	if err != nil {
		return errors.Wrap(err, "failed to delete existing rules")
	}
	ms.log.Infof("Deleted %v pre-existing audit rules.", n)

	// Add rules from config.
	var failCount int
	for _, rule := range rules {
		if err = client.AddRule(rule.data); err != nil {
			// Treat rule add errors as warnings and continue.
			err = errors.Wrapf(err, "failed to add audit rule '%v'", rule.flags)
			reporter.Error(err)
			ms.log.Warnw("Failure adding audit rule", "error", err)
			failCount++
		}
	}
	ms.log.Infof("Successfully added %d of %d audit rules.",
		len(rules)-failCount, len(rules))
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
	ms.log.Infow("audit status from kernel at start", "audit_status", status)

	if status.Enabled == auditLocked {
		return errors.New("failed to configure: The audit system is locked")
	}

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
	if err := ms.client.WaitForPendingACKs(); err != nil {
		return errors.Wrap(err, "failed to wait for ACKs")
	}
	if err := ms.client.SetPID(libaudit.WaitForReply); err != nil {
		if errno, ok := err.(syscall.Errno); ok && errno == syscall.EEXIST && status.PID != 0 {
			return fmt.Errorf("failed to set audit PID. An audit process is already running (PID %d)", status.PID)
		}
		return errors.Wrapf(err, "failed to set audit PID (current audit PID %d)", status.PID)
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
				ms.log.Debugw("Dropping audit message",
					"record_type", raw.Type,
					"message", string(raw.Data),
					"error", err)
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

func buildMetricbeatEvent(msgs []*auparse.AuditMessage, config Config) mb.Event {
	auditEvent, err := aucoalesce.CoalesceMessages(msgs)
	if err != nil {
		// Add messages on error so that it's possible to debug the problem.
		out := mb.Event{MetricSetFields: common.MapStr{}}
		addMessages(msgs, out.MetricSetFields)
		return out
	}

	if config.ResolveIDs {
		aucoalesce.ResolveIDs(auditEvent)
	}

	out := mb.Event{
		Timestamp: auditEvent.Timestamp,
		RootFields: common.MapStr{
			"event": common.MapStr{
				"category": auditEvent.Category.String(),
				"type":     strings.ToLower(auditEvent.Type.String()),
				"action":   auditEvent.Summary.Action,
			},
		},
		ModuleFields: common.MapStr{
			"sequence": auditEvent.Sequence,
			"result":   auditEvent.Result,
			"session":  auditEvent.Session,
			"data":     createAuditdData(auditEvent.Data),
		},
	}

	// Add root level fields.
	addUser(auditEvent.User, out.RootFields)
	addProcess(auditEvent.Process, out.RootFields)
	addFile(auditEvent.File, out.RootFields)
	addAddress(auditEvent.Source, "source", out.RootFields)
	addAddress(auditEvent.Dest, "destination", out.RootFields)
	addNetwork(auditEvent.Net, out.RootFields)
	if len(auditEvent.Tags) > 0 {
		out.RootFields.Put("tags", auditEvent.Tags)
	}

	// Add module fields.
	m := out.ModuleFields
	if auditEvent.Summary.Actor.Primary != "" {
		m.Put("summary.actor.primary", auditEvent.Summary.Actor.Primary)
	}
	if auditEvent.Summary.Actor.Secondary != "" {
		m.Put("summary.actor.secondary", auditEvent.Summary.Actor.Secondary)
	}
	if auditEvent.Summary.Object.Primary != "" {
		m.Put("summary.object.primary", auditEvent.Summary.Object.Primary)
	}
	if auditEvent.Summary.Object.Secondary != "" {
		m.Put("summary.object.secondary", auditEvent.Summary.Object.Secondary)
	}
	if auditEvent.Summary.Object.Type != "" {
		m.Put("summary.object.type", auditEvent.Summary.Object.Type)
	}
	if auditEvent.Summary.How != "" {
		m.Put("summary.how", auditEvent.Summary.How)
	}
	if len(auditEvent.Paths) > 0 {
		m.Put("paths", auditEvent.Paths)
	}
	if config.Warnings && len(auditEvent.Warnings) > 0 {
		warnings := make([]string, 0, len(auditEvent.Warnings))
		for _, err := range auditEvent.Warnings {
			warnings = append(warnings, err.Error())
		}
		m.Put("warnings", warnings)
		addMessages(msgs, m)
	}
	if config.RawMessage {
		addMessages(msgs, m)
	}

	return out
}

func addUser(u aucoalesce.User, m common.MapStr) {
	user := make(common.MapStr, len(u.IDs))
	m.Put("user", user)

	for id, value := range u.IDs {
		user[id] = value
		if len(u.SELinux) > 0 {
			user["selinux"] = u.SELinux
		}
		if len(u.Names) > 0 {
			user["name_map"] = u.Names
		}
	}
}

func addProcess(p aucoalesce.Process, m common.MapStr) {
	if p.IsEmpty() {
		return
	}

	process := common.MapStr{}
	m.Put("process", process)
	if p.PID != "" {
		process["pid"] = p.PID
	}
	if p.PPID != "" {
		process["ppid"] = p.PPID
	}
	if p.Title != "" {
		process["title"] = p.Title
	}
	if p.Name != "" {
		process["name"] = p.Name
	}
	if p.Exe != "" {
		process["exe"] = p.Exe
	}
	if p.CWD != "" {
		process["cwd"] = p.CWD
	}
	if len(p.Args) > 0 {
		process["args"] = p.Args
	}
}

func addFile(f *aucoalesce.File, m common.MapStr) {
	if f == nil {
		return
	}

	file := common.MapStr{}
	m.Put("file", file)
	if f.Path != "" {
		file["path"] = f.Path
	}
	if f.Device != "" {
		file["device"] = f.Device
	}
	if f.Inode != "" {
		file["inode"] = f.Inode
	}
	if f.Mode != "" {
		file["mode"] = f.Mode
	}
	if f.UID != "" {
		file["uid"] = f.UID
	}
	if f.GID != "" {
		file["gid"] = f.GID
	}
	if f.Owner != "" {
		file["owner"] = f.Owner
	}
	if f.Group != "" {
		file["group"] = f.Group
	}
	if len(f.SELinux) > 0 {
		file["selinux"] = f.SELinux
	}
}

func addAddress(addr *aucoalesce.Address, key string, m common.MapStr) {
	if addr == nil {
		return
	}

	address := common.MapStr{}
	m.Put(key, address)
	if addr.Hostname != "" {
		address["hostname"] = addr.Hostname
	}
	if addr.IP != "" {
		address["ip"] = addr.IP
	}
	if addr.Port != "" {
		address["port"] = addr.Port
	}
	if addr.Path != "" {
		address["path"] = addr.Path
	}
}

func addNetwork(net *aucoalesce.Network, m common.MapStr) {
	if net == nil {
		return
	}

	network := common.MapStr{
		"direction": net.Direction,
	}
	m.Put("network", network)
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

func createAuditdData(data map[string]string) common.MapStr {
	out := make(common.MapStr, len(data))
	for key, v := range data {
		if strings.HasPrefix(key, "socket_") {
			out.Put("socket."+key[7:], v)
			continue
		}

		out.Put(key, v)
	}
	return out
}

// stream type

// stream receives callbacks from the libaudit.Reassembler for completed events
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

	length := len(uname.Release)
	data := make([]byte, length)
	for i, v := range uname.Release {
		if v == 0 {
			length = i
			break
		}
		data[i] = byte(v)
	}

	release := string(data[:length])
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

func determineSocketType(c *Config, log *logp.Logger) (string, error) {
	client, err := libaudit.NewAuditClient(nil)
	if err != nil {
		if c.SocketType == "" {
			return "", errors.Wrap(err, "failed to create audit client")
		}
		// Ignore errors if a socket type has been specified. It will fail during
		// further setup and its necessary for unit tests to pass
		return c.SocketType, nil
	}
	defer client.Close()
	status, err := client.GetStatus()
	if err != nil {
		if c.SocketType == "" {
			return "", errors.Wrap(err, "failed to get audit status")
		}
		return c.SocketType, nil
	}
	rules, _ := c.rules()

	isLocked := status.Enabled == auditLocked
	hasMulticast := hasMulticastSupport()
	hasRules := len(rules) > 0

	const useAutodetect = "Remove the socket_type option to have auditbeat " +
		"select the most suitable subscription method."
	switch c.SocketType {
	case unicast:
		if isLocked {
			log.Errorf("requested unicast socket_type is not available "+
				"because audit configuration is locked in the kernel "+
				"(enabled=2). %s", useAutodetect)
			return "", errors.New("unicast socket_type not available")
		}
		return c.SocketType, nil

	case multicast:
		if hasMulticast {
			if hasRules {
				log.Warn("The audit rules specified in the configuration " +
					"cannot be applied when using a multicast socket_type.")
			}
			return c.SocketType, nil
		}
		log.Errorf("socket_type is set to multicast but based on the "+
			"kernel version, multicast audit subscriptions are not supported. %s",
			useAutodetect)
		return "", errors.New("multicast socket_type not available")

	default:
		// attempt to determine the optimal socket_type
		if hasMulticast {
			if hasRules {
				if isLocked {
					log.Warn("Audit rules specified in the configuration " +
						"cannot be applied because the audit rules have been locked " +
						"in the kernel (enabled=2). A multicast audit subscription " +
						"will be used instead, which does not support setting rules")
					return multicast, nil
				}
				return unicast, nil
			}
			return multicast, nil
		}
		if isLocked {
			log.Errorf("Cannot continue: audit configuration is locked " +
				"in the kernel (enabled=2) which prevents using unicast " +
				"sockets. Multicast audit subscriptions are not available " +
				"in this kernel. Disable locking the audit configuration " +
				"to use auditbeat.")
			return "", errors.New("no connection to audit available")
		}
		return unicast, nil
	}

}
