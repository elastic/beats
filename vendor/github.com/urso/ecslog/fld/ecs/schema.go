package ecs

import (
	"time"

	"github.com/urso/ecslog/fld"
)

type (
	nsAgent struct {
	}

	nsClient struct {
	}

	nsCloud struct {
		Account nsCloudAccount

		Instance nsCloudInstance

		Machine nsCloudMachine
	}

	nsCloudAccount struct {
	}

	nsCloudInstance struct {
	}

	nsCloudMachine struct {
	}

	nsContainer struct {
		Image nsContainerImage
	}

	nsContainerImage struct {
	}

	nsDestination struct {
	}

	nsError struct {
	}

	nsEvent struct {
	}

	nsFile struct {
	}

	nsGeo struct {
	}

	nsGroup struct {
	}

	nsHost struct {
	}

	nsHTTP struct {
		Request nsHTTPRequest

		Response nsHTTPResponse
	}

	nsHTTPRequest struct {
		Body nsHTTPRequestBody
	}

	nsHTTPRequestBody struct {
	}

	nsHTTPResponse struct {
		Body nsHTTPResponseBody
	}

	nsHTTPResponseBody struct {
	}

	nsLog struct {
	}

	nsNetwork struct {
	}

	nsObserver struct {
	}

	nsOrganization struct {
	}

	nsOS struct {
	}

	nsProcess struct {
		Thread nsProcessThread
	}

	nsProcessThread struct {
	}

	nsRelated struct {
	}

	nsServer struct {
	}

	nsService struct {
	}

	nsSource struct {
	}

	nsURL struct {
	}

	nsUser struct {
	}

	nsUserAgent struct {
		Device nsUserAgentDevice
	}

	nsUserAgentDevice struct {
	}
)

var (

	// Agent provides fields in the ECS agent namespace.
	// The agent fields contain the data about the software entity, if any,
	// that collects, detects, or observes events on a host, or takes
	// measurements on a host. Examples include Beats. Agents may also run on
	// observers. ECS agent.* fields shall be populated with details of the
	// agent running on the host or observer where the event happened or the
	// measurement was taken.
	Agent = nsAgent{}

	// Client provides fields in the ECS client namespace.
	// A client is defined as the initiator of a network connection for events
	// regarding sessions, connections, or bidirectional flow records. For TCP
	// events, the client is the initiator of the TCP connection that sends
	// the SYN packet(s). For other protocols, the client is generally the
	// initiator or requestor in the network transaction. Some systems use the
	// term "originator" to refer the client in TCP connections. The client
	// fields describe details about the system acting as the client in the
	// network event. Client fields are usually populated in conjunction with
	// server fields. Client fields are generally not populated for
	// packet-level events. Client / server representations can add semantic
	// context to an exchange, which is helpful to visualize the data in
	// certain situations. If your context falls in that category, you should
	// still ensure that source and destination are filled appropriately.
	Client = nsClient{}

	// Cloud provides fields in the ECS cloud namespace.
	// Fields related to the cloud or infrastructure the events are coming
	// from.
	Cloud = nsCloud{}

	// Container provides fields in the ECS container namespace.
	// Container fields are used for meta information about the specific
	// container that is the source of information. These fields help
	// correlate data based containers from any runtime.
	Container = nsContainer{}

	// Destination provides fields in the ECS destination namespace.
	// Destination fields describe details about the destination of a
	// packet/event. Destination fields are usually populated in conjunction
	// with source fields.
	Destination = nsDestination{}

	// Error provides fields in the ECS error namespace.
	// These fields can represent errors of any kind. Use them for errors that
	// happen while fetching events or in cases where the event itself
	// contains an error.
	Error = nsError{}

	// Event provides fields in the ECS event namespace.
	// The event fields are used for context information about the log or
	// metric event itself. A log is defined as an event containing details of
	// something that happened. Log events must include the time at which the
	// thing happened. Examples of log events include a process starting on a
	// host, a network packet being sent from a source to a destination, or a
	// network connection between a client and a server being initiated or
	// closed. A metric is defined as an event containing one or more
	// numerical or categorical measurements and the time at which the
	// measurement was taken. Examples of metric events include memory
	// pressure measured on a host, or vulnerabilities measured on a scanned
	// host.
	Event = nsEvent{}

	// File provides fields in the ECS file namespace.
	// A file is defined as a set of information that has been created on, or
	// has existed on a filesystem. File objects can be associated with host
	// events, network events, and/or file events (e.g., those produced by
	// File Integrity Monitoring [FIM] products or services). File fields
	// provide details about the affected file associated with the event or
	// metric.
	File = nsFile{}

	// Geo provides fields in the ECS geo namespace.
	// Geo fields can carry data about a specific location related to an
	// event. This geolocation information can be derived from techniques such
	// as Geo IP, or be user-supplied.
	Geo = nsGeo{}

	// Group provides fields in the ECS group namespace.
	// The group fields are meant to represent groups that are relevant to the
	// event.
	Group = nsGroup{}

	// Host provides fields in the ECS host namespace.
	// A host is defined as a general computing instance. ECS host.* fields
	// should be populated with details about the host on which the event
	// happened, or from which the measurement was taken. Host types include
	// hardware, virtual machines, Docker containers, and Kubernetes nodes.
	Host = nsHost{}

	// HTTP provides fields in the ECS http namespace.
	// Fields related to HTTP activity. Use the `url` field set to store the
	// url of the request.
	HTTP = nsHTTP{}

	// Log provides fields in the ECS log namespace.
	// Fields which are specific to log events.
	Log = nsLog{}

	// Network provides fields in the ECS network namespace.
	// The network is defined as the communication path over which a host or
	// network event happens. The network.* fields should be populated with
	// details about the network activity associated with an event.
	Network = nsNetwork{}

	// Observer provides fields in the ECS observer namespace.
	// An observer is defined as a special network, security, or application
	// device used to detect, observe, or create network, security, or
	// application-related events and metrics. This could be a custom hardware
	// appliance or a server that has been configured to run special network,
	// security, or application software. Examples include firewalls,
	// intrusion detection/prevention systems, network monitoring sensors, web
	// application firewalls, data loss prevention systems, and APM servers.
	// The observer.* fields shall be populated with details of the system, if
	// any, that detects, observes and/or creates a network, security, or
	// application event or metric. Message queues and ETL components used in
	// processing events or metrics are not considered observers in ECS.
	Observer = nsObserver{}

	// Organization provides fields in the ECS organization namespace.
	// The organization fields enrich data with information about the company
	// or entity the data is associated with. These fields help you arrange or
	// filter data stored in an index by one or multiple organizations.
	Organization = nsOrganization{}

	// OS provides fields in the ECS os namespace.
	// The OS fields contain information about the operating system.
	OS = nsOS{}

	// Process provides fields in the ECS process namespace.
	// These fields contain information about a process. These fields can help
	// you correlate metrics information with a process id/name from a log
	// message.  The `process.pid` often stays in the metric itself and is
	// copied to the global field for correlation.
	Process = nsProcess{}

	// Related provides fields in the ECS related namespace.
	// This field set is meant to facilitate pivoting around a piece of data.
	// Some pieces of information can be seen in many places in an ECS event.
	// To facilitate searching for them, store an array of all seen values to
	// their corresponding field in `related.`. A concrete example is IP
	// addresses, which can be under host, observer, source, destination,
	// client, server, and network.forwarded_ip. If you append all IPs to
	// `related.ip`, you can then search for a given IP trivially, no matter
	// where it appeared, by querying `related.ip:a.b.c.d`.
	Related = nsRelated{}

	// Server provides fields in the ECS server namespace.
	// A Server is defined as the responder in a network connection for events
	// regarding sessions, connections, or bidirectional flow records. For TCP
	// events, the server is the receiver of the initial SYN packet(s) of the
	// TCP connection. For other protocols, the server is generally the
	// responder in the network transaction. Some systems actually use the
	// term "responder" to refer the server in TCP connections. The server
	// fields describe details about the system acting as the server in the
	// network event. Server fields are usually populated in conjunction with
	// client fields. Server fields are generally not populated for
	// packet-level events. Client / server representations can add semantic
	// context to an exchange, which is helpful to visualize the data in
	// certain situations. If your context falls in that category, you should
	// still ensure that source and destination are filled appropriately.
	Server = nsServer{}

	// Service provides fields in the ECS service namespace.
	// The service fields describe the service for or from which the data was
	// collected. These fields help you find and correlate logs for a specific
	// service and version.
	Service = nsService{}

	// Source provides fields in the ECS source namespace.
	// Source fields describe details about the source of a packet/event.
	// Source fields are usually populated in conjunction with destination
	// fields.
	Source = nsSource{}

	// URL provides fields in the ECS url namespace.
	// URL fields provide support for complete or partial URLs, and supports
	// the breaking down into scheme, domain, path, and so on.
	URL = nsURL{}

	// User provides fields in the ECS user namespace.
	// The user fields describe information about the user that is relevant to
	// the event. Fields can have one entry or multiple entries. If a user has
	// more than one id, provide an array that includes all of them.
	User = nsUser{}

	// UserAgent provides fields in the ECS user_agent namespace.
	// The user_agent fields normally come from a browser request. They often
	// show up in web service logs coming from the parsed user agent string.
	UserAgent = nsUserAgent{}
)

const Version = "1.0.0"

func ecsField(key string, val fld.Value) fld.Field {
	return fld.Field{Key: key, Value: val, Standardized: true}
}

func ecsAny(key string, val interface{}) fld.Field   { return ecsField(key, fld.ValAny(val)) }
func ecsTime(key string, val time.Time) fld.Field    { return ecsField(key, fld.ValTime(val)) }
func ecsDur(key string, val time.Duration) fld.Field { return ecsField(key, fld.ValDuration(val)) }
func ecsString(key, val string) fld.Field            { return ecsField(key, fld.ValString(val)) }
func ecsInt(key string, val int) fld.Field           { return ecsField(key, fld.ValInt(val)) }
func ecsInt64(key string, val int64) fld.Field       { return ecsField(key, fld.ValInt64(val)) }
func ecsFloat64(key string, val float64) fld.Field   { return ecsField(key, fld.ValFloat(val)) }

// ## agent fields

// EphemeralID create the ECS complain 'agent.ephemeral_id' field.
// Ephemeral identifier of this agent (if one exists). This id normally
// changes across restarts, but `agent.id` does not.
func (nsAgent) EphemeralID(value string) fld.Field {
	return ecsString("agent.ephemeral_id", value)
}

// Name create the ECS complain 'agent.name' field.
// Custom name of the agent. This is a name that can be given to an agent.
// This can be helpful if for example two Filebeat instances are running
// on the same host but a human readable separation is needed on which
// Filebeat instance data is coming from. If no name is given, the name is
// often left empty.
func (nsAgent) Name(value string) fld.Field {
	return ecsString("agent.name", value)
}

// Type create the ECS complain 'agent.type' field.
// Type of the agent. The agent type stays always the same and should be
// given by the agent used. In case of Filebeat the agent would always be
// Filebeat also if two Filebeat instances are run on the same machine.
func (nsAgent) Type(value string) fld.Field {
	return ecsString("agent.type", value)
}

// ID create the ECS complain 'agent.id' field.
// Unique identifier of this agent (if one exists). Example: For Beats
// this would be beat.id.
func (nsAgent) ID(value string) fld.Field {
	return ecsString("agent.id", value)
}

// Version create the ECS complain 'agent.version' field.
// Version of the agent.
func (nsAgent) Version(value string) fld.Field {
	return ecsString("agent.version", value)
}

// ## client fields

// Packets create the ECS complain 'client.packets' field.
// Packets sent from the client to the server.
func (nsClient) Packets(value int64) fld.Field {
	return ecsInt64("client.packets", value)
}

// Port create the ECS complain 'client.port' field.
// Port of the client.
func (nsClient) Port(value int64) fld.Field {
	return ecsInt64("client.port", value)
}

// MAC create the ECS complain 'client.mac' field.
// MAC address of the client.
func (nsClient) MAC(value string) fld.Field {
	return ecsString("client.mac", value)
}

// Domain create the ECS complain 'client.domain' field.
// Client domain.
func (nsClient) Domain(value string) fld.Field {
	return ecsString("client.domain", value)
}

// Bytes create the ECS complain 'client.bytes' field.
// Bytes sent from the client to the server.
func (nsClient) Bytes(value int64) fld.Field {
	return ecsInt64("client.bytes", value)
}

// IP create the ECS complain 'client.ip' field.
// IP address of the client. Can be one or multiple IPv4 or IPv6
// addresses.
func (nsClient) IP(value string) fld.Field {
	return ecsString("client.ip", value)
}

// Address create the ECS complain 'client.address' field.
// Some event client addresses are defined ambiguously. The event will
// sometimes list an IP, a domain or a unix socket.  You should always
// store the raw address in the `.address` field. Then it should be
// duplicated to `.ip` or `.domain`, depending on which one it is.
func (nsClient) Address(value string) fld.Field {
	return ecsString("client.address", value)
}

// ## cloud fields

// Region create the ECS complain 'cloud.region' field.
// Region in which this host is running.
func (nsCloud) Region(value string) fld.Field {
	return ecsString("cloud.region", value)
}

// AvailabilityZone create the ECS complain 'cloud.availability_zone' field.
// Availability zone in which this host is running.
func (nsCloud) AvailabilityZone(value string) fld.Field {
	return ecsString("cloud.availability_zone", value)
}

// Provider create the ECS complain 'cloud.provider' field.
// Name of the cloud provider. Example values are aws, azure, gcp, or
// digitalocean.
func (nsCloud) Provider(value string) fld.Field {
	return ecsString("cloud.provider", value)
}

// ## cloud.account fields

// ID create the ECS complain 'cloud.account.id' field.
// The cloud account or organization id used to identify different
// entities in a multi-tenant environment. Examples: AWS account id,
// Google Cloud ORG Id, or other unique identifier.
func (nsCloudAccount) ID(value string) fld.Field {
	return ecsString("cloud.account.id", value)
}

// ## cloud.instance fields

// Name create the ECS complain 'cloud.instance.name' field.
// Instance name of the host machine.
func (nsCloudInstance) Name(value string) fld.Field {
	return ecsString("cloud.instance.name", value)
}

// ID create the ECS complain 'cloud.instance.id' field.
// Instance ID of the host machine.
func (nsCloudInstance) ID(value string) fld.Field {
	return ecsString("cloud.instance.id", value)
}

// ## cloud.machine fields

// Type create the ECS complain 'cloud.machine.type' field.
// Machine type of the host machine.
func (nsCloudMachine) Type(value string) fld.Field {
	return ecsString("cloud.machine.type", value)
}

// ## container fields

// Labels create the ECS complain 'container.labels' field.
// Image labels.
func (nsContainer) Labels(value map[string]interface{}) fld.Field {
	return ecsAny("container.labels", value)
}

// ID create the ECS complain 'container.id' field.
// Unique container id.
func (nsContainer) ID(value string) fld.Field {
	return ecsString("container.id", value)
}

// Name create the ECS complain 'container.name' field.
// Container name.
func (nsContainer) Name(value string) fld.Field {
	return ecsString("container.name", value)
}

// Runtime create the ECS complain 'container.runtime' field.
// Runtime managing this container.
func (nsContainer) Runtime(value string) fld.Field {
	return ecsString("container.runtime", value)
}

// ## container.image fields

// Tag create the ECS complain 'container.image.tag' field.
// Container image tag.
func (nsContainerImage) Tag(value string) fld.Field {
	return ecsString("container.image.tag", value)
}

// Name create the ECS complain 'container.image.name' field.
// Name of the image the container was built on.
func (nsContainerImage) Name(value string) fld.Field {
	return ecsString("container.image.name", value)
}

// ## destination fields

// Packets create the ECS complain 'destination.packets' field.
// Packets sent from the destination to the source.
func (nsDestination) Packets(value int64) fld.Field {
	return ecsInt64("destination.packets", value)
}

// Domain create the ECS complain 'destination.domain' field.
// Destination domain.
func (nsDestination) Domain(value string) fld.Field {
	return ecsString("destination.domain", value)
}

// IP create the ECS complain 'destination.ip' field.
// IP address of the destination. Can be one or multiple IPv4 or IPv6
// addresses.
func (nsDestination) IP(value string) fld.Field {
	return ecsString("destination.ip", value)
}

// MAC create the ECS complain 'destination.mac' field.
// MAC address of the destination.
func (nsDestination) MAC(value string) fld.Field {
	return ecsString("destination.mac", value)
}

// Address create the ECS complain 'destination.address' field.
// Some event destination addresses are defined ambiguously. The event
// will sometimes list an IP, a domain or a unix socket.  You should
// always store the raw address in the `.address` field. Then it should be
// duplicated to `.ip` or `.domain`, depending on which one it is.
func (nsDestination) Address(value string) fld.Field {
	return ecsString("destination.address", value)
}

// Port create the ECS complain 'destination.port' field.
// Port of the destination.
func (nsDestination) Port(value int64) fld.Field {
	return ecsInt64("destination.port", value)
}

// Bytes create the ECS complain 'destination.bytes' field.
// Bytes sent from the destination to the source.
func (nsDestination) Bytes(value int64) fld.Field {
	return ecsInt64("destination.bytes", value)
}

// ## error fields

// Message create the ECS complain 'error.message' field.
// Error message.
func (nsError) Message(value string) fld.Field {
	return ecsString("error.message", value)
}

// Code create the ECS complain 'error.code' field.
// Error code describing the error.
func (nsError) Code(value string) fld.Field {
	return ecsString("error.code", value)
}

// ID create the ECS complain 'error.id' field.
// Unique identifier for the error.
func (nsError) ID(value string) fld.Field {
	return ecsString("error.id", value)
}

// ## event fields

// Duration create the ECS complain 'event.duration' field.
// Duration of the event in nanoseconds. If event.start and event.end are
// known this value should be the difference between the end and start
// time.
func (nsEvent) Duration(value int64) fld.Field {
	return ecsInt64("event.duration", value)
}

// Module create the ECS complain 'event.module' field.
// Name of the module this data is coming from. This information is coming
// from the modules used in Beats or Logstash.
func (nsEvent) Module(value string) fld.Field {
	return ecsString("event.module", value)
}

// Timezone create the ECS complain 'event.timezone' field.
// This field should be populated when the event's timestamp does not
// include timezone information already (e.g. default Syslog timestamps).
// It's optional otherwise. Acceptable timezone formats are: a canonical
// ID (e.g. "Europe/Amsterdam"), abbreviated (e.g. "EST") or an HH:mm
// differential (e.g. "-05:00").
func (nsEvent) Timezone(value string) fld.Field {
	return ecsString("event.timezone", value)
}

// Outcome create the ECS complain 'event.outcome' field.
// The outcome of the event. If the event describes an action, this fields
// contains the outcome of that action. Examples outcomes are `success`
// and `failure`. Warning: In future versions of ECS, we plan to provide a
// list of acceptable values for this field, please use with caution.
func (nsEvent) Outcome(value string) fld.Field {
	return ecsString("event.outcome", value)
}

// ID create the ECS complain 'event.id' field.
// Unique ID to describe the event.
func (nsEvent) ID(value string) fld.Field {
	return ecsString("event.id", value)
}

// Hash create the ECS complain 'event.hash' field.
// Hash (perhaps logstash fingerprint) of raw field to be able to
// demonstrate log integrity.
func (nsEvent) Hash(value string) fld.Field {
	return ecsString("event.hash", value)
}

// Action create the ECS complain 'event.action' field.
// The action captured by the event. This describes the information in the
// event. It is more specific than `event.category`. Examples are
// `group-add`, `process-started`, `file-created`. The value is normally
// defined by the implementer.
func (nsEvent) Action(value string) fld.Field {
	return ecsString("event.action", value)
}

// Original create the ECS complain 'event.original' field.
// Raw text message of entire event. Used to demonstrate log integrity.
// This field is not indexed and doc_values are disabled. It cannot be
// searched, but it can be retrieved from `_source`.
func (nsEvent) Original(value string) fld.Field {
	return ecsString("event.original", value)
}

// Category create the ECS complain 'event.category' field.
// Event category. This contains high-level information about the contents
// of the event. It is more generic than `event.action`, in the sense that
// typically a category contains multiple actions. Warning: In future
// versions of ECS, we plan to provide a list of acceptable values for
// this field, please use with caution.
func (nsEvent) Category(value string) fld.Field {
	return ecsString("event.category", value)
}

// Severity create the ECS complain 'event.severity' field.
// Severity describes the original severity of the event. What the
// different severity values mean can very different between use cases.
// It's up to the implementer to make sure severities are consistent
// across events.
func (nsEvent) Severity(value int64) fld.Field {
	return ecsInt64("event.severity", value)
}

// RiskScoreNorm create the ECS complain 'event.risk_score_norm' field.
// Normalized risk score or priority of the event, on a scale of 0 to 100.
// This is mainly useful if you use more than one system that assigns risk
// scores, and you want to see a normalized value across all systems.
func (nsEvent) RiskScoreNorm(value float64) fld.Field {
	return ecsFloat64("event.risk_score_norm", value)
}

// Kind create the ECS complain 'event.kind' field.
// The kind of the event. This gives information about what type of
// information the event contains, without being specific to the contents
// of the event.  Examples are `event`, `state`, `alarm`. Warning: In
// future versions of ECS, we plan to provide a list of acceptable values
// for this field, please use with caution.
func (nsEvent) Kind(value string) fld.Field {
	return ecsString("event.kind", value)
}

// End create the ECS complain 'event.end' field.
// event.end contains the date when the event ended or when the activity
// was last observed.
func (nsEvent) End(value time.Time) fld.Field {
	return ecsTime("event.end", value)
}

// Type create the ECS complain 'event.type' field.
// Reserved for future usage. Please avoid using this field for user data.
func (nsEvent) Type(value string) fld.Field {
	return ecsString("event.type", value)
}

// Created create the ECS complain 'event.created' field.
// event.created contains the date/time when the event was first read by
// an agent, or by your pipeline. This field is distinct from @timestamp
// in that @timestamp typically contain the time extracted from the
// original event. In most situations, these two timestamps will be
// slightly different. The difference can be used to calculate the delay
// between your source generating an event, and the time when your agent
// first processed it. This can be used to monitor your agent's or
// pipeline's ability to keep up with your event source. In case the two
// timestamps are identical, @timestamp should be used.
func (nsEvent) Created(value time.Time) fld.Field {
	return ecsTime("event.created", value)
}

// RiskScore create the ECS complain 'event.risk_score' field.
// Risk score or priority of the event (e.g. security solutions). Use your
// system's original value here.
func (nsEvent) RiskScore(value float64) fld.Field {
	return ecsFloat64("event.risk_score", value)
}

// Dataset create the ECS complain 'event.dataset' field.
// Name of the dataset. The concept of a `dataset` (fileset / metricset)
// is used in Beats as a subset of modules. It contains the information
// which is currently stored in metricset.name and metricset.module or
// fileset.name.
func (nsEvent) Dataset(value string) fld.Field {
	return ecsString("event.dataset", value)
}

// Start create the ECS complain 'event.start' field.
// event.start contains the date when the event started or when the
// activity was first observed.
func (nsEvent) Start(value time.Time) fld.Field {
	return ecsTime("event.start", value)
}

// ## file fields

// Mode create the ECS complain 'file.mode' field.
// Mode of the file in octal representation.
func (nsFile) Mode(value string) fld.Field {
	return ecsString("file.mode", value)
}

// Mtime create the ECS complain 'file.mtime' field.
// Last time file content was modified.
func (nsFile) Mtime(value time.Time) fld.Field {
	return ecsTime("file.mtime", value)
}

// Inode create the ECS complain 'file.inode' field.
// Inode representing the file in the filesystem.
func (nsFile) Inode(value string) fld.Field {
	return ecsString("file.inode", value)
}

// Device create the ECS complain 'file.device' field.
// Device that is the source of the file.
func (nsFile) Device(value string) fld.Field {
	return ecsString("file.device", value)
}

// Path create the ECS complain 'file.path' field.
// Path to the file.
func (nsFile) Path(value string) fld.Field {
	return ecsString("file.path", value)
}

// Type create the ECS complain 'file.type' field.
// File type (file, dir, or symlink).
func (nsFile) Type(value string) fld.Field {
	return ecsString("file.type", value)
}

// Gid create the ECS complain 'file.gid' field.
// Primary group ID (GID) of the file.
func (nsFile) Gid(value string) fld.Field {
	return ecsString("file.gid", value)
}

// TargetPath create the ECS complain 'file.target_path' field.
// Target path for symlinks.
func (nsFile) TargetPath(value string) fld.Field {
	return ecsString("file.target_path", value)
}

// Size create the ECS complain 'file.size' field.
// File size in bytes (field is only added when `type` is `file`).
func (nsFile) Size(value int64) fld.Field {
	return ecsInt64("file.size", value)
}

// Extension create the ECS complain 'file.extension' field.
// File extension. This should allow easy filtering by file extensions.
func (nsFile) Extension(value string) fld.Field {
	return ecsString("file.extension", value)
}

// Group create the ECS complain 'file.group' field.
// Primary group name of the file.
func (nsFile) Group(value string) fld.Field {
	return ecsString("file.group", value)
}

// Ctime create the ECS complain 'file.ctime' field.
// Last time file metadata changed.
func (nsFile) Ctime(value time.Time) fld.Field {
	return ecsTime("file.ctime", value)
}

// Owner create the ECS complain 'file.owner' field.
// File owner's username.
func (nsFile) Owner(value string) fld.Field {
	return ecsString("file.owner", value)
}

// UID create the ECS complain 'file.uid' field.
// The user ID (UID) or security identifier (SID) of the file owner.
func (nsFile) UID(value string) fld.Field {
	return ecsString("file.uid", value)
}

// ## geo fields

// Location create the ECS complain 'geo.location' field.
// Longitude and latitude.
func (nsGeo) Location(value string) fld.Field {
	return ecsString("geo.location", value)
}

// ContinentName create the ECS complain 'geo.continent_name' field.
// Name of the continent.
func (nsGeo) ContinentName(value string) fld.Field {
	return ecsString("geo.continent_name", value)
}

// CountryName create the ECS complain 'geo.country_name' field.
// Country name.
func (nsGeo) CountryName(value string) fld.Field {
	return ecsString("geo.country_name", value)
}

// Name create the ECS complain 'geo.name' field.
// User-defined description of a location, at the level of granularity
// they care about. Could be the name of their data centers, the floor
// number, if this describes a local physical entity, city names. Not
// typically used in automated geolocation.
func (nsGeo) Name(value string) fld.Field {
	return ecsString("geo.name", value)
}

// CountryIsoCode create the ECS complain 'geo.country_iso_code' field.
// Country ISO code.
func (nsGeo) CountryIsoCode(value string) fld.Field {
	return ecsString("geo.country_iso_code", value)
}

// RegionIsoCode create the ECS complain 'geo.region_iso_code' field.
// Region ISO code.
func (nsGeo) RegionIsoCode(value string) fld.Field {
	return ecsString("geo.region_iso_code", value)
}

// CityName create the ECS complain 'geo.city_name' field.
// City name.
func (nsGeo) CityName(value string) fld.Field {
	return ecsString("geo.city_name", value)
}

// RegionName create the ECS complain 'geo.region_name' field.
// Region name.
func (nsGeo) RegionName(value string) fld.Field {
	return ecsString("geo.region_name", value)
}

// ## group fields

// Name create the ECS complain 'group.name' field.
// Name of the group.
func (nsGroup) Name(value string) fld.Field {
	return ecsString("group.name", value)
}

// ID create the ECS complain 'group.id' field.
// Unique identifier for the group on the system/platform.
func (nsGroup) ID(value string) fld.Field {
	return ecsString("group.id", value)
}

// ## host fields

// ID create the ECS complain 'host.id' field.
// Unique host id. As hostname is not always unique, use values that are
// meaningful in your environment. Example: The current usage of
// `beat.name`.
func (nsHost) ID(value string) fld.Field {
	return ecsString("host.id", value)
}

// IP create the ECS complain 'host.ip' field.
// Host ip address.
func (nsHost) IP(value string) fld.Field {
	return ecsString("host.ip", value)
}

// MAC create the ECS complain 'host.mac' field.
// Host mac address.
func (nsHost) MAC(value string) fld.Field {
	return ecsString("host.mac", value)
}

// Architecture create the ECS complain 'host.architecture' field.
// Operating system architecture.
func (nsHost) Architecture(value string) fld.Field {
	return ecsString("host.architecture", value)
}

// Name create the ECS complain 'host.name' field.
// Name of the host. It can contain what `hostname` returns on Unix
// systems, the fully qualified domain name, or a name specified by the
// user. The sender decides which value to use.
func (nsHost) Name(value string) fld.Field {
	return ecsString("host.name", value)
}

// Type create the ECS complain 'host.type' field.
// Type of host. For Cloud providers this can be the machine type like
// `t2.medium`. If vm, this could be the container, for example, or other
// information meaningful in your environment.
func (nsHost) Type(value string) fld.Field {
	return ecsString("host.type", value)
}

// Hostname create the ECS complain 'host.hostname' field.
// Hostname of the host. It normally contains what the `hostname` command
// returns on the host machine.
func (nsHost) Hostname(value string) fld.Field {
	return ecsString("host.hostname", value)
}

// ## http fields

// Version create the ECS complain 'http.version' field.
// HTTP version.
func (nsHTTP) Version(value string) fld.Field {
	return ecsString("http.version", value)
}

// ## http.request fields

// Referrer create the ECS complain 'http.request.referrer' field.
// Referrer for this HTTP request.
func (nsHTTPRequest) Referrer(value string) fld.Field {
	return ecsString("http.request.referrer", value)
}

// Bytes create the ECS complain 'http.request.bytes' field.
// Total size in bytes of the request (body and headers).
func (nsHTTPRequest) Bytes(value int64) fld.Field {
	return ecsInt64("http.request.bytes", value)
}

// Method create the ECS complain 'http.request.method' field.
// HTTP request method. The field value must be normalized to lowercase
// for querying. See the documentation section "Implementing ECS".
func (nsHTTPRequest) Method(value string) fld.Field {
	return ecsString("http.request.method", value)
}

// ## http.request.body fields

// Bytes create the ECS complain 'http.request.body.bytes' field.
// Size in bytes of the request body.
func (nsHTTPRequestBody) Bytes(value int64) fld.Field {
	return ecsInt64("http.request.body.bytes", value)
}

// Content create the ECS complain 'http.request.body.content' field.
// The full HTTP request body.
func (nsHTTPRequestBody) Content(value string) fld.Field {
	return ecsString("http.request.body.content", value)
}

// ## http.response fields

// StatusCode create the ECS complain 'http.response.status_code' field.
// HTTP response status code.
func (nsHTTPResponse) StatusCode(value int64) fld.Field {
	return ecsInt64("http.response.status_code", value)
}

// Bytes create the ECS complain 'http.response.bytes' field.
// Total size in bytes of the response (body and headers).
func (nsHTTPResponse) Bytes(value int64) fld.Field {
	return ecsInt64("http.response.bytes", value)
}

// ## http.response.body fields

// Content create the ECS complain 'http.response.body.content' field.
// The full HTTP response body.
func (nsHTTPResponseBody) Content(value string) fld.Field {
	return ecsString("http.response.body.content", value)
}

// Bytes create the ECS complain 'http.response.body.bytes' field.
// Size in bytes of the response body.
func (nsHTTPResponseBody) Bytes(value int64) fld.Field {
	return ecsInt64("http.response.body.bytes", value)
}

// ## log fields

// Level create the ECS complain 'log.level' field.
// Original log level of the log event. Some examples are `warn`, `error`,
// `i`.
func (nsLog) Level(value string) fld.Field {
	return ecsString("log.level", value)
}

// Original create the ECS complain 'log.original' field.
// This is the original log message and contains the full log message
// before splitting it up in multiple parts. In contrast to the `message`
// field which can contain an extracted part of the log message, this
// field contains the original, full log message. It can have already some
// modifications applied like encoding or new lines removed to clean up
// the log message. This field is not indexed and doc_values are disabled
// so it can't be queried but the value can be retrieved from `_source`.
func (nsLog) Original(value string) fld.Field {
	return ecsString("log.original", value)
}

// ## network fields

// Bytes create the ECS complain 'network.bytes' field.
// Total bytes transferred in both directions. If `source.bytes` and
// `destination.bytes` are known, `network.bytes` is their sum.
func (nsNetwork) Bytes(value int64) fld.Field {
	return ecsInt64("network.bytes", value)
}

// CommunityID create the ECS complain 'network.community_id' field.
// A hash of source and destination IPs and ports, as well as the protocol
// used in a communication. This is a tool-agnostic standard to identify
// flows. Learn more at https://github.com/corelight/community-id-spec.
func (nsNetwork) CommunityID(value string) fld.Field {
	return ecsString("network.community_id", value)
}

// Type create the ECS complain 'network.type' field.
// In the OSI Model this would be the Network Layer. ipv4, ipv6, ipsec,
// pim, etc The field value must be normalized to lowercase for querying.
// See the documentation section "Implementing ECS".
func (nsNetwork) Type(value string) fld.Field {
	return ecsString("network.type", value)
}

// ForwardedIP create the ECS complain 'network.forwarded_ip' field.
// Host IP address when the source IP address is the proxy.
func (nsNetwork) ForwardedIP(value string) fld.Field {
	return ecsString("network.forwarded_ip", value)
}

// Name create the ECS complain 'network.name' field.
// Name given by operators to sections of their network.
func (nsNetwork) Name(value string) fld.Field {
	return ecsString("network.name", value)
}

// Application create the ECS complain 'network.application' field.
// A name given to an application level protocol. This can be arbitrarily
// assigned for things like microservices, but also apply to things like
// skype, icq, facebook, twitter. This would be used in situations where
// the vendor or service can be decoded such as from the source/dest IP
// owners, ports, or wire format. The field value must be normalized to
// lowercase for querying. See the documentation section "Implementing
// ECS".
func (nsNetwork) Application(value string) fld.Field {
	return ecsString("network.application", value)
}

// Packets create the ECS complain 'network.packets' field.
// Total packets transferred in both directions. If `source.packets` and
// `destination.packets` are known, `network.packets` is their sum.
func (nsNetwork) Packets(value int64) fld.Field {
	return ecsInt64("network.packets", value)
}

// Protocol create the ECS complain 'network.protocol' field.
// L7 Network protocol name. ex. http, lumberjack, transport protocol. The
// field value must be normalized to lowercase for querying. See the
// documentation section "Implementing ECS".
func (nsNetwork) Protocol(value string) fld.Field {
	return ecsString("network.protocol", value)
}

// IANANumber create the ECS complain 'network.iana_number' field.
// IANA Protocol Number
// (https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml).
// Standardized list of protocols. This aligns well with NetFlow and sFlow
// related logs which use the IANA Protocol Number.
func (nsNetwork) IANANumber(value string) fld.Field {
	return ecsString("network.iana_number", value)
}

// Transport create the ECS complain 'network.transport' field.
// Same as network.iana_number, but instead using the Keyword name of the
// transport layer (udp, tcp, ipv6-icmp, etc.) The field value must be
// normalized to lowercase for querying. See the documentation section
// "Implementing ECS".
func (nsNetwork) Transport(value string) fld.Field {
	return ecsString("network.transport", value)
}

// Direction create the ECS complain 'network.direction' field.
// Direction of the network traffic. Recommended values are:   * inbound
// * outbound   * internal   * external   * unknown  When mapping events
// from a host-based monitoring context, populate this field from the
// host's point of view. When mapping events from a network or
// perimeter-based monitoring context, populate this field from the point
// of view of your network perimeter.
func (nsNetwork) Direction(value string) fld.Field {
	return ecsString("network.direction", value)
}

// ## observer fields

// IP create the ECS complain 'observer.ip' field.
// IP address of the observer.
func (nsObserver) IP(value string) fld.Field {
	return ecsString("observer.ip", value)
}

// Hostname create the ECS complain 'observer.hostname' field.
// Hostname of the observer.
func (nsObserver) Hostname(value string) fld.Field {
	return ecsString("observer.hostname", value)
}

// MAC create the ECS complain 'observer.mac' field.
// MAC address of the observer
func (nsObserver) MAC(value string) fld.Field {
	return ecsString("observer.mac", value)
}

// SerialNumber create the ECS complain 'observer.serial_number' field.
// Observer serial number.
func (nsObserver) SerialNumber(value string) fld.Field {
	return ecsString("observer.serial_number", value)
}

// Type create the ECS complain 'observer.type' field.
// The type of the observer the data is coming from. There is no
// predefined list of observer types. Some examples are `forwarder`,
// `firewall`, `ids`, `ips`, `proxy`, `poller`, `sensor`, `APM server`.
func (nsObserver) Type(value string) fld.Field {
	return ecsString("observer.type", value)
}

// Vendor create the ECS complain 'observer.vendor' field.
// observer vendor information.
func (nsObserver) Vendor(value string) fld.Field {
	return ecsString("observer.vendor", value)
}

// Version create the ECS complain 'observer.version' field.
// Observer version.
func (nsObserver) Version(value string) fld.Field {
	return ecsString("observer.version", value)
}

// ## organization fields

// Name create the ECS complain 'organization.name' field.
// Organization name.
func (nsOrganization) Name(value string) fld.Field {
	return ecsString("organization.name", value)
}

// ID create the ECS complain 'organization.id' field.
// Unique identifier for the organization.
func (nsOrganization) ID(value string) fld.Field {
	return ecsString("organization.id", value)
}

// ## os fields

// Platform create the ECS complain 'os.platform' field.
// Operating system platform (such centos, ubuntu, windows).
func (nsOS) Platform(value string) fld.Field {
	return ecsString("os.platform", value)
}

// Name create the ECS complain 'os.name' field.
// Operating system name, without the version.
func (nsOS) Name(value string) fld.Field {
	return ecsString("os.name", value)
}

// Full create the ECS complain 'os.full' field.
// Operating system name, including the version or code name.
func (nsOS) Full(value string) fld.Field {
	return ecsString("os.full", value)
}

// Kernel create the ECS complain 'os.kernel' field.
// Operating system kernel version as a raw string.
func (nsOS) Kernel(value string) fld.Field {
	return ecsString("os.kernel", value)
}

// Family create the ECS complain 'os.family' field.
// OS family (such as redhat, debian, freebsd, windows).
func (nsOS) Family(value string) fld.Field {
	return ecsString("os.family", value)
}

// Version create the ECS complain 'os.version' field.
// Operating system version as a raw string.
func (nsOS) Version(value string) fld.Field {
	return ecsString("os.version", value)
}

// ## process fields

// Start create the ECS complain 'process.start' field.
// The time the process started.
func (nsProcess) Start(value time.Time) fld.Field {
	return ecsTime("process.start", value)
}

// PPID create the ECS complain 'process.ppid' field.
// Process parent id.
func (nsProcess) PPID(value int64) fld.Field {
	return ecsInt64("process.ppid", value)
}

// Args create the ECS complain 'process.args' field.
// Array of process arguments. May be filtered to protect sensitive
// information.
func (nsProcess) Args(value string) fld.Field {
	return ecsString("process.args", value)
}

// PID create the ECS complain 'process.pid' field.
// Process id.
func (nsProcess) PID(value int64) fld.Field {
	return ecsInt64("process.pid", value)
}

// Name create the ECS complain 'process.name' field.
// Process name. Sometimes called program name or similar.
func (nsProcess) Name(value string) fld.Field {
	return ecsString("process.name", value)
}

// WorkingDirectory create the ECS complain 'process.working_directory' field.
// The working directory of the process.
func (nsProcess) WorkingDirectory(value string) fld.Field {
	return ecsString("process.working_directory", value)
}

// Title create the ECS complain 'process.title' field.
// Process title. The proctitle, some times the same as process name. Can
// also be different: for example a browser setting its title to the web
// page currently opened.
func (nsProcess) Title(value string) fld.Field {
	return ecsString("process.title", value)
}

// Executable create the ECS complain 'process.executable' field.
// Absolute path to the process executable.
func (nsProcess) Executable(value string) fld.Field {
	return ecsString("process.executable", value)
}

// ## process.thread fields

// ID create the ECS complain 'process.thread.id' field.
// Thread ID.
func (nsProcessThread) ID(value int64) fld.Field {
	return ecsInt64("process.thread.id", value)
}

// ## related fields

// IP create the ECS complain 'related.ip' field.
// All of the IPs seen on your event.
func (nsRelated) IP(value string) fld.Field {
	return ecsString("related.ip", value)
}

// ## server fields

// IP create the ECS complain 'server.ip' field.
// IP address of the server. Can be one or multiple IPv4 or IPv6
// addresses.
func (nsServer) IP(value string) fld.Field {
	return ecsString("server.ip", value)
}

// Address create the ECS complain 'server.address' field.
// Some event server addresses are defined ambiguously. The event will
// sometimes list an IP, a domain or a unix socket.  You should always
// store the raw address in the `.address` field. Then it should be
// duplicated to `.ip` or `.domain`, depending on which one it is.
func (nsServer) Address(value string) fld.Field {
	return ecsString("server.address", value)
}

// Port create the ECS complain 'server.port' field.
// Port of the server.
func (nsServer) Port(value int64) fld.Field {
	return ecsInt64("server.port", value)
}

// MAC create the ECS complain 'server.mac' field.
// MAC address of the server.
func (nsServer) MAC(value string) fld.Field {
	return ecsString("server.mac", value)
}

// Bytes create the ECS complain 'server.bytes' field.
// Bytes sent from the server to the client.
func (nsServer) Bytes(value int64) fld.Field {
	return ecsInt64("server.bytes", value)
}

// Packets create the ECS complain 'server.packets' field.
// Packets sent from the server to the client.
func (nsServer) Packets(value int64) fld.Field {
	return ecsInt64("server.packets", value)
}

// Domain create the ECS complain 'server.domain' field.
// Server domain.
func (nsServer) Domain(value string) fld.Field {
	return ecsString("server.domain", value)
}

// ## service fields

// Name create the ECS complain 'service.name' field.
// Name of the service data is collected from. The name of the service is
// normally user given. This allows if two instances of the same service
// are running on the same machine they can be differentiated by the
// `service.name`. Also it allows for distributed services that run on
// multiple hosts to correlate the related instances based on the name. In
// the case of Elasticsearch the service.name could contain the cluster
// name. For Beats the service.name is by default a copy of the
// `service.type` field if no name is specified.
func (nsService) Name(value string) fld.Field {
	return ecsString("service.name", value)
}

// Type create the ECS complain 'service.type' field.
// The type of the service data is collected from. The type can be used to
// group and correlate logs and metrics from one service type. Example: If
// logs or metrics are collected from Elasticsearch, `service.type` would
// be `elasticsearch`.
func (nsService) Type(value string) fld.Field {
	return ecsString("service.type", value)
}

// Version create the ECS complain 'service.version' field.
// Version of the service the data was collected from. This allows to look
// at a data set only for a specific version of a service.
func (nsService) Version(value string) fld.Field {
	return ecsString("service.version", value)
}

// ID create the ECS complain 'service.id' field.
// Unique identifier of the running service. This id should uniquely
// identify this service. This makes it possible to correlate logs and
// metrics for one specific service. Example: If you are experiencing
// issues with one redis instance, you can filter on that id to see
// metrics and logs for that single instance.
func (nsService) ID(value string) fld.Field {
	return ecsString("service.id", value)
}

// EphemeralID create the ECS complain 'service.ephemeral_id' field.
// Ephemeral identifier of this service (if one exists). This id normally
// changes across restarts, but `service.id` does not.
func (nsService) EphemeralID(value string) fld.Field {
	return ecsString("service.ephemeral_id", value)
}

// State create the ECS complain 'service.state' field.
// Current state of the service.
func (nsService) State(value string) fld.Field {
	return ecsString("service.state", value)
}

// ## source fields

// MAC create the ECS complain 'source.mac' field.
// MAC address of the source.
func (nsSource) MAC(value string) fld.Field {
	return ecsString("source.mac", value)
}

// Address create the ECS complain 'source.address' field.
// Some event source addresses are defined ambiguously. The event will
// sometimes list an IP, a domain or a unix socket.  You should always
// store the raw address in the `.address` field. Then it should be
// duplicated to `.ip` or `.domain`, depending on which one it is.
func (nsSource) Address(value string) fld.Field {
	return ecsString("source.address", value)
}

// Domain create the ECS complain 'source.domain' field.
// Source domain.
func (nsSource) Domain(value string) fld.Field {
	return ecsString("source.domain", value)
}

// Bytes create the ECS complain 'source.bytes' field.
// Bytes sent from the source to the destination.
func (nsSource) Bytes(value int64) fld.Field {
	return ecsInt64("source.bytes", value)
}

// Packets create the ECS complain 'source.packets' field.
// Packets sent from the source to the destination.
func (nsSource) Packets(value int64) fld.Field {
	return ecsInt64("source.packets", value)
}

// Port create the ECS complain 'source.port' field.
// Port of the source.
func (nsSource) Port(value int64) fld.Field {
	return ecsInt64("source.port", value)
}

// IP create the ECS complain 'source.ip' field.
// IP address of the source. Can be one or multiple IPv4 or IPv6
// addresses.
func (nsSource) IP(value string) fld.Field {
	return ecsString("source.ip", value)
}

// ## url fields

// Query create the ECS complain 'url.query' field.
// The query field describes the query string of the request, such as
// "q=elasticsearch". The `?` is excluded from the query string. If a URL
// contains no `?`, there is no query field. If there is a `?` but no
// query, the query field exists with an empty string. The `exists` query
// can be used to differentiate between the two cases.
func (nsURL) Query(value string) fld.Field {
	return ecsString("url.query", value)
}

// Port create the ECS complain 'url.port' field.
// Port of the request, such as 443.
func (nsURL) Port(value int64) fld.Field {
	return ecsInt64("url.port", value)
}

// Domain create the ECS complain 'url.domain' field.
// Domain of the url, such as "www.elastic.co". In some cases a URL may
// refer to an IP and/or port directly, without a domain name. In this
// case, the IP address would go to the `domain` field.
func (nsURL) Domain(value string) fld.Field {
	return ecsString("url.domain", value)
}

// Full create the ECS complain 'url.full' field.
// If full URLs are important to your use case, they should be stored in
// `url.full`, whether this field is reconstructed or present in the event
// source.
func (nsURL) Full(value string) fld.Field {
	return ecsString("url.full", value)
}

// Username create the ECS complain 'url.username' field.
// Username of the request.
func (nsURL) Username(value string) fld.Field {
	return ecsString("url.username", value)
}

// Password create the ECS complain 'url.password' field.
// Password of the request.
func (nsURL) Password(value string) fld.Field {
	return ecsString("url.password", value)
}

// Scheme create the ECS complain 'url.scheme' field.
// Scheme of the request, such as "https". Note: The `:` is not part of
// the scheme.
func (nsURL) Scheme(value string) fld.Field {
	return ecsString("url.scheme", value)
}

// Path create the ECS complain 'url.path' field.
// Path of the request, such as "/search".
func (nsURL) Path(value string) fld.Field {
	return ecsString("url.path", value)
}

// Original create the ECS complain 'url.original' field.
// Unmodified original url as seen in the event source. Note that in
// network monitoring, the observed URL may be a full URL, whereas in
// access logs, the URL is often just represented as a path. This field is
// meant to represent the URL as it was observed, complete or not.
func (nsURL) Original(value string) fld.Field {
	return ecsString("url.original", value)
}

// Fragment create the ECS complain 'url.fragment' field.
// Portion of the url after the `#`, such as "top". The `#` is not part of
// the fragment.
func (nsURL) Fragment(value string) fld.Field {
	return ecsString("url.fragment", value)
}

// ## user fields

// Name create the ECS complain 'user.name' field.
// Short name or login of the user.
func (nsUser) Name(value string) fld.Field {
	return ecsString("user.name", value)
}

// Hash create the ECS complain 'user.hash' field.
// Unique user hash to correlate information for a user in anonymized
// form. Useful if `user.id` or `user.name` contain confidential
// information and cannot be used.
func (nsUser) Hash(value string) fld.Field {
	return ecsString("user.hash", value)
}

// Email create the ECS complain 'user.email' field.
// User email address.
func (nsUser) Email(value string) fld.Field {
	return ecsString("user.email", value)
}

// FullName create the ECS complain 'user.full_name' field.
// User's full name, if available.
func (nsUser) FullName(value string) fld.Field {
	return ecsString("user.full_name", value)
}

// ID create the ECS complain 'user.id' field.
// One or multiple unique identifiers of the user.
func (nsUser) ID(value string) fld.Field {
	return ecsString("user.id", value)
}

// ## user_agent fields

// Original create the ECS complain 'user_agent.original' field.
// Unparsed version of the user_agent.
func (nsUserAgent) Original(value string) fld.Field {
	return ecsString("user_agent.original", value)
}

// Version create the ECS complain 'user_agent.version' field.
// Version of the user agent.
func (nsUserAgent) Version(value string) fld.Field {
	return ecsString("user_agent.version", value)
}

// Name create the ECS complain 'user_agent.name' field.
// Name of the user agent.
func (nsUserAgent) Name(value string) fld.Field {
	return ecsString("user_agent.name", value)
}

// ## user_agent.device fields

// Name create the ECS complain 'user_agent.device.name' field.
// Name of the device.
func (nsUserAgentDevice) Name(value string) fld.Field {
	return ecsString("user_agent.device.name", value)
}
