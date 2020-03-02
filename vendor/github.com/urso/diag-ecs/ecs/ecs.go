package ecs

import (
	"time"

	"github.com/urso/diag"
)

type (
	nsAgent struct {
	}

	nsAs struct {
		Organization nsAsOrganization
	}

	nsAsOrganization struct {
	}

	nsClient struct {
		Geo nsClientGeo

		User nsClientUser

		Nat nsClientNat

		As nsClientAs
	}

	nsClientAs struct {
		Organization nsClientAsOrganization
	}

	nsClientAsOrganization struct {
	}

	nsClientGeo struct {
	}

	nsClientNat struct {
	}

	nsClientUser struct {
		Group nsClientUserGroup
	}

	nsClientUserGroup struct {
	}

	nsCloud struct {
		Instance nsCloudInstance

		Account nsCloudAccount

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
		User nsDestinationUser

		Geo nsDestinationGeo

		Nat nsDestinationNat

		As nsDestinationAs
	}

	nsDestinationAs struct {
		Organization nsDestinationAsOrganization
	}

	nsDestinationAsOrganization struct {
	}

	nsDestinationGeo struct {
	}

	nsDestinationNat struct {
	}

	nsDestinationUser struct {
		Group nsDestinationUserGroup
	}

	nsDestinationUserGroup struct {
	}

	nsDNS struct {
		Question nsDNSQuestion

		Answers nsDNSAnswers
	}

	nsDNSAnswers struct {
	}

	nsDNSQuestion struct {
	}

	nsECS struct {
	}

	nsError struct {
	}

	nsEvent struct {
	}

	nsFile struct {
		Hash nsFileHash
	}

	nsFileHash struct {
	}

	nsGeo struct {
	}

	nsGroup struct {
	}

	nsHash struct {
	}

	nsHost struct {
		OS nsHostOS

		User nsHostUser

		Geo nsHostGeo
	}

	nsHostGeo struct {
	}

	nsHostOS struct {
	}

	nsHostUser struct {
		Group nsHostUserGroup
	}

	nsHostUserGroup struct {
	}

	nsHTTP struct {
		Response nsHTTPResponse

		Request nsHTTPRequest
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
		Origin nsLogOrigin

		Syslog nsLogSyslog
	}

	nsLogOrigin struct {
		File nsLogOriginFile
	}

	nsLogOriginFile struct {
	}

	nsLogSyslog struct {
		Severity nsLogSyslogSeverity

		Facility nsLogSyslogFacility
	}

	nsLogSyslogFacility struct {
	}

	nsLogSyslogSeverity struct {
	}

	nsNetwork struct {
	}

	nsObserver struct {
		Geo nsObserverGeo

		OS nsObserverOS
	}

	nsObserverGeo struct {
	}

	nsObserverOS struct {
	}

	nsOrganization struct {
	}

	nsOS struct {
	}

	nsPackage struct {
	}

	nsProcess struct {
		Parent nsProcessParent

		Hash nsProcessHash

		Thread nsProcessThread
	}

	nsProcessHash struct {
	}

	nsProcessParent struct {
		Thread nsProcessParentThread
	}

	nsProcessParentThread struct {
	}

	nsProcessThread struct {
	}

	nsRegistry struct {
		Data nsRegistryData
	}

	nsRegistryData struct {
	}

	nsRelated struct {
	}

	nsRule struct {
	}

	nsServer struct {
		User nsServerUser

		Nat nsServerNat

		Geo nsServerGeo

		As nsServerAs
	}

	nsServerAs struct {
		Organization nsServerAsOrganization
	}

	nsServerAsOrganization struct {
	}

	nsServerGeo struct {
	}

	nsServerNat struct {
	}

	nsServerUser struct {
		Group nsServerUserGroup
	}

	nsServerUserGroup struct {
	}

	nsService struct {
		Node nsServiceNode
	}

	nsServiceNode struct {
	}

	nsSource struct {
		As nsSourceAs

		Geo nsSourceGeo

		User nsSourceUser

		Nat nsSourceNat
	}

	nsSourceAs struct {
		Organization nsSourceAsOrganization
	}

	nsSourceAsOrganization struct {
	}

	nsSourceGeo struct {
	}

	nsSourceNat struct {
	}

	nsSourceUser struct {
		Group nsSourceUserGroup
	}

	nsSourceUserGroup struct {
	}

	nsThreat struct {
		Tactic nsThreatTactic

		Technique nsThreatTechnique
	}

	nsThreatTactic struct {
	}

	nsThreatTechnique struct {
	}

	nsTLS struct {
		Client nsTLSClient

		Server nsTLSServer
	}

	nsTLSClient struct {
		Hash nsTLSClientHash
	}

	nsTLSClientHash struct {
	}

	nsTLSServer struct {
		Hash nsTLSServerHash
	}

	nsTLSServerHash struct {
	}

	nsTrace struct {
	}

	nsTransaction struct {
	}

	nsURL struct {
	}

	nsUser struct {
		Group nsUserGroup
	}

	nsUserGroup struct {
	}

	nsUserAgent struct {
		OS nsUserAgentOS

		Device nsUserAgentDevice
	}

	nsUserAgentDevice struct {
	}

	nsUserAgentOS struct {
	}

	nsVulnerability struct {
		Score nsVulnerabilityScore

		Scanner nsVulnerabilityScanner
	}

	nsVulnerabilityScanner struct {
	}

	nsVulnerabilityScore struct {
	}
)

var (

	// Agent provides fields in the ECS agent namespace.
	Agent = nsAgent{}

	// As provides fields in the ECS as namespace.
	As = nsAs{}

	// Client provides fields in the ECS client namespace.
	Client = nsClient{}

	// Cloud provides fields in the ECS cloud namespace.
	Cloud = nsCloud{}

	// Container provides fields in the ECS container namespace.
	Container = nsContainer{}

	// Destination provides fields in the ECS destination namespace.
	Destination = nsDestination{}

	// DNS provides fields in the ECS dns namespace.
	DNS = nsDNS{}

	// ECS provides fields in the ECS ecs namespace.
	ECS = nsECS{}

	// Error provides fields in the ECS error namespace.
	Error = nsError{}

	// Event provides fields in the ECS event namespace.
	Event = nsEvent{}

	// File provides fields in the ECS file namespace.
	File = nsFile{}

	// Geo provides fields in the ECS geo namespace.
	Geo = nsGeo{}

	// Group provides fields in the ECS group namespace.
	Group = nsGroup{}

	// Hash provides fields in the ECS hash namespace.
	Hash = nsHash{}

	// Host provides fields in the ECS host namespace.
	Host = nsHost{}

	// HTTP provides fields in the ECS http namespace.
	HTTP = nsHTTP{}

	// Log provides fields in the ECS log namespace.
	Log = nsLog{}

	// Network provides fields in the ECS network namespace.
	Network = nsNetwork{}

	// Observer provides fields in the ECS observer namespace.
	Observer = nsObserver{}

	// Organization provides fields in the ECS organization namespace.
	Organization = nsOrganization{}

	// OS provides fields in the ECS os namespace.
	OS = nsOS{}

	// Package provides fields in the ECS package namespace.
	Package = nsPackage{}

	// Process provides fields in the ECS process namespace.
	Process = nsProcess{}

	// Registry provides fields in the ECS registry namespace.
	Registry = nsRegistry{}

	// Related provides fields in the ECS related namespace.
	Related = nsRelated{}

	// Rule provides fields in the ECS rule namespace.
	Rule = nsRule{}

	// Server provides fields in the ECS server namespace.
	Server = nsServer{}

	// Service provides fields in the ECS service namespace.
	Service = nsService{}

	// Source provides fields in the ECS source namespace.
	Source = nsSource{}

	// Threat provides fields in the ECS threat namespace.
	Threat = nsThreat{}

	// TLS provides fields in the ECS tls namespace.
	TLS = nsTLS{}

	// Trace provides fields in the ECS trace namespace.
	Trace = nsTrace{}

	// Transaction provides fields in the ECS transaction namespace.
	Transaction = nsTransaction{}

	// URL provides fields in the ECS url namespace.
	URL = nsURL{}

	// User provides fields in the ECS user namespace.
	User = nsUser{}

	// UserAgent provides fields in the ECS user_agent namespace.
	UserAgent = nsUserAgent{}

	// Vulnerability provides fields in the ECS vulnerability namespace.
	Vulnerability = nsVulnerability{}
)

// Version is the current ECS version available in the ecs package.
const Version = "1.4.0"

func ecsField(key string, val diag.Value) diag.Field {
	return diag.Field{Key: key, Value: val, Standardized: true}
}

func ecsAny(key string, val interface{}) diag.Field   { return ecsField(key, diag.ValAny(val)) }
func ecsTime(key string, val time.Time) diag.Field    { return ecsField(key, diag.ValTime(val)) }
func ecsDur(key string, val time.Duration) diag.Field { return ecsField(key, diag.ValDuration(val)) }
func ecsString(key, val string) diag.Field            { return ecsField(key, diag.ValString(val)) }
func ecsBool(key string, val bool) diag.Field         { return ecsField(key, diag.ValBool(val)) }
func ecsInt(key string, val int) diag.Field           { return ecsField(key, diag.ValInt(val)) }
func ecsInt64(key string, val int64) diag.Field       { return ecsField(key, diag.ValInt64(val)) }
func ecsFloat64(key string, val float64) diag.Field   { return ecsField(key, diag.ValFloat(val)) }

// Timestamp create the ECS complain '@timestamp' field.
// Date/time when the event originated. This is the date/time extracted
// from the event, typically representing when the event was generated by
// the source. If the event source has no original timestamp, this value
// is typically populated by the first time the event was received by the
// pipeline. Required field for all events.
func Timestamp(value time.Time) diag.Field {
	return ecsTime("@timestamp", value)
}

// Message create the ECS complain 'message' field.
// For log events the message field contains the log message, optimized
// for viewing in a log viewer. For structured logs without an original
// message field, other fields can be concatenated to form a
// human-readable summary of the event. If multiple messages exist, they
// can be combined into one message.
func Message(value string) diag.Field {
	return ecsString("message", value)
}

// ## agent fields

// ID create the ECS complain 'agent.id' field.
// Unique identifier of this agent (if one exists). Example: For Beats
// this would be beat.id.
func (nsAgent) ID(value string) diag.Field {
	return ecsString("agent.id", value)
}

// Name create the ECS complain 'agent.name' field.
// Custom name of the agent. This is a name that can be given to an agent.
// This can be helpful if for example two Filebeat instances are running
// on the same host but a human readable separation is needed on which
// Filebeat instance data is coming from. If no name is given, the name is
// often left empty.
func (nsAgent) Name(value string) diag.Field {
	return ecsString("agent.name", value)
}

// EphemeralID create the ECS complain 'agent.ephemeral_id' field.
// Ephemeral identifier of this agent (if one exists). This id normally
// changes across restarts, but `agent.id` does not.
func (nsAgent) EphemeralID(value string) diag.Field {
	return ecsString("agent.ephemeral_id", value)
}

// Version create the ECS complain 'agent.version' field.
// Version of the agent.
func (nsAgent) Version(value string) diag.Field {
	return ecsString("agent.version", value)
}

// Type create the ECS complain 'agent.type' field.
// Type of the agent. The agent type stays always the same and should be
// given by the agent used. In case of Filebeat the agent would always be
// Filebeat also if two Filebeat instances are run on the same machine.
func (nsAgent) Type(value string) diag.Field {
	return ecsString("agent.type", value)
}

// ## as fields

// Number create the ECS complain 'as.number' field.
// Unique number allocated to the autonomous system. The autonomous system
// number (ASN) uniquely identifies each network on the Internet.
func (nsAs) Number(value int64) diag.Field {
	return ecsInt64("as.number", value)
}

// ## as.organization fields

// Name create the ECS complain 'as.organization.name' field.
// Organization name.
func (nsAsOrganization) Name(value string) diag.Field {
	return ecsString("as.organization.name", value)
}

// ## client fields

// MAC create the ECS complain 'client.mac' field.
// MAC address of the client.
func (nsClient) MAC(value string) diag.Field {
	return ecsString("client.mac", value)
}

// TopLevelDomain create the ECS complain 'client.top_level_domain' field.
// The effective top level domain (eTLD), also known as the domain suffix,
// is the last part of the domain name. For example, the top level domain
// for google.com is "com". This value can be determined precisely with a
// list like the public suffix list (http://publicsuffix.org). Trying to
// approximate this by simply taking the last label will not work well for
// effective TLDs such as "co.uk".
func (nsClient) TopLevelDomain(value string) diag.Field {
	return ecsString("client.top_level_domain", value)
}

// Domain create the ECS complain 'client.domain' field.
// Client domain.
func (nsClient) Domain(value string) diag.Field {
	return ecsString("client.domain", value)
}

// Port create the ECS complain 'client.port' field.
// Port of the client.
func (nsClient) Port(value int64) diag.Field {
	return ecsInt64("client.port", value)
}

// Address create the ECS complain 'client.address' field.
// Some event client addresses are defined ambiguously. The event will
// sometimes list an IP, a domain or a unix socket.  You should always
// store the raw address in the `.address` field. Then it should be
// duplicated to `.ip` or `.domain`, depending on which one it is.
func (nsClient) Address(value string) diag.Field {
	return ecsString("client.address", value)
}

// Bytes create the ECS complain 'client.bytes' field.
// Bytes sent from the client to the server.
func (nsClient) Bytes(value int64) diag.Field {
	return ecsInt64("client.bytes", value)
}

// IP create the ECS complain 'client.ip' field.
// IP address of the client. Can be one or multiple IPv4 or IPv6
// addresses.
func (nsClient) IP(value string) diag.Field {
	return ecsString("client.ip", value)
}

// Packets create the ECS complain 'client.packets' field.
// Packets sent from the client to the server.
func (nsClient) Packets(value int64) diag.Field {
	return ecsInt64("client.packets", value)
}

// RegisteredDomain create the ECS complain 'client.registered_domain' field.
// The highest registered client domain, stripped of the subdomain. For
// example, the registered domain for "foo.google.com" is "google.com".
// This value can be determined precisely with a list like the public
// suffix list (http://publicsuffix.org). Trying to approximate this by
// simply taking the last two labels will not work well for TLDs such as
// "co.uk".
func (nsClient) RegisteredDomain(value string) diag.Field {
	return ecsString("client.registered_domain", value)
}

// ## client.as fields

// Number create the ECS complain 'client.as.number' field.
// Unique number allocated to the autonomous system. The autonomous system
// number (ASN) uniquely identifies each network on the Internet.
func (nsClientAs) Number(value int64) diag.Field {
	return ecsInt64("client.as.number", value)
}

// ## client.as.organization fields

// Name create the ECS complain 'client.as.organization.name' field.
// Organization name.
func (nsClientAsOrganization) Name(value string) diag.Field {
	return ecsString("client.as.organization.name", value)
}

// ## client.geo fields

// Name create the ECS complain 'client.geo.name' field.
// User-defined description of a location, at the level of granularity
// they care about. Could be the name of their data centers, the floor
// number, if this describes a local physical entity, city names. Not
// typically used in automated geolocation.
func (nsClientGeo) Name(value string) diag.Field {
	return ecsString("client.geo.name", value)
}

// ContinentName create the ECS complain 'client.geo.continent_name' field.
// Name of the continent.
func (nsClientGeo) ContinentName(value string) diag.Field {
	return ecsString("client.geo.continent_name", value)
}

// RegionName create the ECS complain 'client.geo.region_name' field.
// Region name.
func (nsClientGeo) RegionName(value string) diag.Field {
	return ecsString("client.geo.region_name", value)
}

// CountryIsoCode create the ECS complain 'client.geo.country_iso_code' field.
// Country ISO code.
func (nsClientGeo) CountryIsoCode(value string) diag.Field {
	return ecsString("client.geo.country_iso_code", value)
}

// RegionIsoCode create the ECS complain 'client.geo.region_iso_code' field.
// Region ISO code.
func (nsClientGeo) RegionIsoCode(value string) diag.Field {
	return ecsString("client.geo.region_iso_code", value)
}

// CountryName create the ECS complain 'client.geo.country_name' field.
// Country name.
func (nsClientGeo) CountryName(value string) diag.Field {
	return ecsString("client.geo.country_name", value)
}

// CityName create the ECS complain 'client.geo.city_name' field.
// City name.
func (nsClientGeo) CityName(value string) diag.Field {
	return ecsString("client.geo.city_name", value)
}

// Location create the ECS complain 'client.geo.location' field.
// Longitude and latitude.
func (nsClientGeo) Location(value string) diag.Field {
	return ecsString("client.geo.location", value)
}

// ## client.nat fields

// IP create the ECS complain 'client.nat.ip' field.
// Translated IP of source based NAT sessions (e.g. internal client to
// internet). Typically connections traversing load balancers, firewalls,
// or routers.
func (nsClientNat) IP(value string) diag.Field {
	return ecsString("client.nat.ip", value)
}

// Port create the ECS complain 'client.nat.port' field.
// Translated port of source based NAT sessions (e.g. internal client to
// internet). Typically connections traversing load balancers, firewalls,
// or routers.
func (nsClientNat) Port(value int64) diag.Field {
	return ecsInt64("client.nat.port", value)
}

// ## client.user fields

// Hash create the ECS complain 'client.user.hash' field.
// Unique user hash to correlate information for a user in anonymized
// form. Useful if `user.id` or `user.name` contain confidential
// information and cannot be used.
func (nsClientUser) Hash(value string) diag.Field {
	return ecsString("client.user.hash", value)
}

// Email create the ECS complain 'client.user.email' field.
// User email address.
func (nsClientUser) Email(value string) diag.Field {
	return ecsString("client.user.email", value)
}

// Name create the ECS complain 'client.user.name' field.
// Short name or login of the user.
func (nsClientUser) Name(value string) diag.Field {
	return ecsString("client.user.name", value)
}

// ID create the ECS complain 'client.user.id' field.
// One or multiple unique identifiers of the user.
func (nsClientUser) ID(value string) diag.Field {
	return ecsString("client.user.id", value)
}

// Domain create the ECS complain 'client.user.domain' field.
// Name of the directory the user is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsClientUser) Domain(value string) diag.Field {
	return ecsString("client.user.domain", value)
}

// FullName create the ECS complain 'client.user.full_name' field.
// User's full name, if available.
func (nsClientUser) FullName(value string) diag.Field {
	return ecsString("client.user.full_name", value)
}

// ## client.user.group fields

// Domain create the ECS complain 'client.user.group.domain' field.
// Name of the directory the group is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsClientUserGroup) Domain(value string) diag.Field {
	return ecsString("client.user.group.domain", value)
}

// ID create the ECS complain 'client.user.group.id' field.
// Unique identifier for the group on the system/platform.
func (nsClientUserGroup) ID(value string) diag.Field {
	return ecsString("client.user.group.id", value)
}

// Name create the ECS complain 'client.user.group.name' field.
// Name of the group.
func (nsClientUserGroup) Name(value string) diag.Field {
	return ecsString("client.user.group.name", value)
}

// ## cloud fields

// AvailabilityZone create the ECS complain 'cloud.availability_zone' field.
// Availability zone in which this host is running.
func (nsCloud) AvailabilityZone(value string) diag.Field {
	return ecsString("cloud.availability_zone", value)
}

// Region create the ECS complain 'cloud.region' field.
// Region in which this host is running.
func (nsCloud) Region(value string) diag.Field {
	return ecsString("cloud.region", value)
}

// Provider create the ECS complain 'cloud.provider' field.
// Name of the cloud provider. Example values are aws, azure, gcp, or
// digitalocean.
func (nsCloud) Provider(value string) diag.Field {
	return ecsString("cloud.provider", value)
}

// ## cloud.account fields

// ID create the ECS complain 'cloud.account.id' field.
// The cloud account or organization id used to identify different
// entities in a multi-tenant environment. Examples: AWS account id,
// Google Cloud ORG Id, or other unique identifier.
func (nsCloudAccount) ID(value string) diag.Field {
	return ecsString("cloud.account.id", value)
}

// ## cloud.instance fields

// ID create the ECS complain 'cloud.instance.id' field.
// Instance ID of the host machine.
func (nsCloudInstance) ID(value string) diag.Field {
	return ecsString("cloud.instance.id", value)
}

// Name create the ECS complain 'cloud.instance.name' field.
// Instance name of the host machine.
func (nsCloudInstance) Name(value string) diag.Field {
	return ecsString("cloud.instance.name", value)
}

// ## cloud.machine fields

// Type create the ECS complain 'cloud.machine.type' field.
// Machine type of the host machine.
func (nsCloudMachine) Type(value string) diag.Field {
	return ecsString("cloud.machine.type", value)
}

// ## container fields

// ID create the ECS complain 'container.id' field.
// Unique container id.
func (nsContainer) ID(value string) diag.Field {
	return ecsString("container.id", value)
}

// Runtime create the ECS complain 'container.runtime' field.
// Runtime managing this container.
func (nsContainer) Runtime(value string) diag.Field {
	return ecsString("container.runtime", value)
}

// Name create the ECS complain 'container.name' field.
// Container name.
func (nsContainer) Name(value string) diag.Field {
	return ecsString("container.name", value)
}

// ## container.image fields

// Name create the ECS complain 'container.image.name' field.
// Name of the image the container was built on.
func (nsContainerImage) Name(value string) diag.Field {
	return ecsString("container.image.name", value)
}

// Tag create the ECS complain 'container.image.tag' field.
// Container image tag.
func (nsContainerImage) Tag(value string) diag.Field {
	return ecsString("container.image.tag", value)
}

// ## destination fields

// IP create the ECS complain 'destination.ip' field.
// IP address of the destination. Can be one or multiple IPv4 or IPv6
// addresses.
func (nsDestination) IP(value string) diag.Field {
	return ecsString("destination.ip", value)
}

// Address create the ECS complain 'destination.address' field.
// Some event destination addresses are defined ambiguously. The event
// will sometimes list an IP, a domain or a unix socket.  You should
// always store the raw address in the `.address` field. Then it should be
// duplicated to `.ip` or `.domain`, depending on which one it is.
func (nsDestination) Address(value string) diag.Field {
	return ecsString("destination.address", value)
}

// Packets create the ECS complain 'destination.packets' field.
// Packets sent from the destination to the source.
func (nsDestination) Packets(value int64) diag.Field {
	return ecsInt64("destination.packets", value)
}

// TopLevelDomain create the ECS complain 'destination.top_level_domain' field.
// The effective top level domain (eTLD), also known as the domain suffix,
// is the last part of the domain name. For example, the top level domain
// for google.com is "com". This value can be determined precisely with a
// list like the public suffix list (http://publicsuffix.org). Trying to
// approximate this by simply taking the last label will not work well for
// effective TLDs such as "co.uk".
func (nsDestination) TopLevelDomain(value string) diag.Field {
	return ecsString("destination.top_level_domain", value)
}

// RegisteredDomain create the ECS complain 'destination.registered_domain' field.
// The highest registered destination domain, stripped of the subdomain.
// For example, the registered domain for "foo.google.com" is
// "google.com". This value can be determined precisely with a list like
// the public suffix list (http://publicsuffix.org). Trying to approximate
// this by simply taking the last two labels will not work well for TLDs
// such as "co.uk".
func (nsDestination) RegisteredDomain(value string) diag.Field {
	return ecsString("destination.registered_domain", value)
}

// Bytes create the ECS complain 'destination.bytes' field.
// Bytes sent from the destination to the source.
func (nsDestination) Bytes(value int64) diag.Field {
	return ecsInt64("destination.bytes", value)
}

// MAC create the ECS complain 'destination.mac' field.
// MAC address of the destination.
func (nsDestination) MAC(value string) diag.Field {
	return ecsString("destination.mac", value)
}

// Port create the ECS complain 'destination.port' field.
// Port of the destination.
func (nsDestination) Port(value int64) diag.Field {
	return ecsInt64("destination.port", value)
}

// Domain create the ECS complain 'destination.domain' field.
// Destination domain.
func (nsDestination) Domain(value string) diag.Field {
	return ecsString("destination.domain", value)
}

// ## destination.as fields

// Number create the ECS complain 'destination.as.number' field.
// Unique number allocated to the autonomous system. The autonomous system
// number (ASN) uniquely identifies each network on the Internet.
func (nsDestinationAs) Number(value int64) diag.Field {
	return ecsInt64("destination.as.number", value)
}

// ## destination.as.organization fields

// Name create the ECS complain 'destination.as.organization.name' field.
// Organization name.
func (nsDestinationAsOrganization) Name(value string) diag.Field {
	return ecsString("destination.as.organization.name", value)
}

// ## destination.geo fields

// ContinentName create the ECS complain 'destination.geo.continent_name' field.
// Name of the continent.
func (nsDestinationGeo) ContinentName(value string) diag.Field {
	return ecsString("destination.geo.continent_name", value)
}

// Name create the ECS complain 'destination.geo.name' field.
// User-defined description of a location, at the level of granularity
// they care about. Could be the name of their data centers, the floor
// number, if this describes a local physical entity, city names. Not
// typically used in automated geolocation.
func (nsDestinationGeo) Name(value string) diag.Field {
	return ecsString("destination.geo.name", value)
}

// RegionIsoCode create the ECS complain 'destination.geo.region_iso_code' field.
// Region ISO code.
func (nsDestinationGeo) RegionIsoCode(value string) diag.Field {
	return ecsString("destination.geo.region_iso_code", value)
}

// CityName create the ECS complain 'destination.geo.city_name' field.
// City name.
func (nsDestinationGeo) CityName(value string) diag.Field {
	return ecsString("destination.geo.city_name", value)
}

// Location create the ECS complain 'destination.geo.location' field.
// Longitude and latitude.
func (nsDestinationGeo) Location(value string) diag.Field {
	return ecsString("destination.geo.location", value)
}

// CountryIsoCode create the ECS complain 'destination.geo.country_iso_code' field.
// Country ISO code.
func (nsDestinationGeo) CountryIsoCode(value string) diag.Field {
	return ecsString("destination.geo.country_iso_code", value)
}

// RegionName create the ECS complain 'destination.geo.region_name' field.
// Region name.
func (nsDestinationGeo) RegionName(value string) diag.Field {
	return ecsString("destination.geo.region_name", value)
}

// CountryName create the ECS complain 'destination.geo.country_name' field.
// Country name.
func (nsDestinationGeo) CountryName(value string) diag.Field {
	return ecsString("destination.geo.country_name", value)
}

// ## destination.nat fields

// Port create the ECS complain 'destination.nat.port' field.
// Port the source session is translated to by NAT Device. Typically used
// with load balancers, firewalls, or routers.
func (nsDestinationNat) Port(value int64) diag.Field {
	return ecsInt64("destination.nat.port", value)
}

// IP create the ECS complain 'destination.nat.ip' field.
// Translated ip of destination based NAT sessions (e.g. internet to
// private DMZ) Typically used with load balancers, firewalls, or routers.
func (nsDestinationNat) IP(value string) diag.Field {
	return ecsString("destination.nat.ip", value)
}

// ## destination.user fields

// Email create the ECS complain 'destination.user.email' field.
// User email address.
func (nsDestinationUser) Email(value string) diag.Field {
	return ecsString("destination.user.email", value)
}

// ID create the ECS complain 'destination.user.id' field.
// One or multiple unique identifiers of the user.
func (nsDestinationUser) ID(value string) diag.Field {
	return ecsString("destination.user.id", value)
}

// Name create the ECS complain 'destination.user.name' field.
// Short name or login of the user.
func (nsDestinationUser) Name(value string) diag.Field {
	return ecsString("destination.user.name", value)
}

// Domain create the ECS complain 'destination.user.domain' field.
// Name of the directory the user is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsDestinationUser) Domain(value string) diag.Field {
	return ecsString("destination.user.domain", value)
}

// FullName create the ECS complain 'destination.user.full_name' field.
// User's full name, if available.
func (nsDestinationUser) FullName(value string) diag.Field {
	return ecsString("destination.user.full_name", value)
}

// Hash create the ECS complain 'destination.user.hash' field.
// Unique user hash to correlate information for a user in anonymized
// form. Useful if `user.id` or `user.name` contain confidential
// information and cannot be used.
func (nsDestinationUser) Hash(value string) diag.Field {
	return ecsString("destination.user.hash", value)
}

// ## destination.user.group fields

// Name create the ECS complain 'destination.user.group.name' field.
// Name of the group.
func (nsDestinationUserGroup) Name(value string) diag.Field {
	return ecsString("destination.user.group.name", value)
}

// Domain create the ECS complain 'destination.user.group.domain' field.
// Name of the directory the group is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsDestinationUserGroup) Domain(value string) diag.Field {
	return ecsString("destination.user.group.domain", value)
}

// ID create the ECS complain 'destination.user.group.id' field.
// Unique identifier for the group on the system/platform.
func (nsDestinationUserGroup) ID(value string) diag.Field {
	return ecsString("destination.user.group.id", value)
}

// ## dns fields

// HeaderFlags create the ECS complain 'dns.header_flags' field.
// Array of 2 letter DNS header flags. Expected values are: AA, TC, RD,
// RA, AD, CD, DO.
func (nsDNS) HeaderFlags(value string) diag.Field {
	return ecsString("dns.header_flags", value)
}

// ResolvedIP create the ECS complain 'dns.resolved_ip' field.
// Array containing all IPs seen in `answers.data`. The `answers` array
// can be difficult to use, because of the variety of data formats it can
// contain. Extracting all IP addresses seen in there to `dns.resolved_ip`
// makes it possible to index them as IP addresses, and makes them easier
// to visualize and query for.
func (nsDNS) ResolvedIP(value string) diag.Field {
	return ecsString("dns.resolved_ip", value)
}

// ResponseCode create the ECS complain 'dns.response_code' field.
// The DNS response code.
func (nsDNS) ResponseCode(value string) diag.Field {
	return ecsString("dns.response_code", value)
}

// Type create the ECS complain 'dns.type' field.
// The type of DNS event captured, query or answer. If your source of DNS
// events only gives you DNS queries, you should only create dns events of
// type `dns.type:query`. If your source of DNS events gives you answers
// as well, you should create one event per query (optionally as soon as
// the query is seen). And a second event containing all query details as
// well as an array of answers.
func (nsDNS) Type(value string) diag.Field {
	return ecsString("dns.type", value)
}

// OpCode create the ECS complain 'dns.op_code' field.
// The DNS operation code that specifies the kind of query in the message.
// This value is set by the originator of a query and copied into the
// response.
func (nsDNS) OpCode(value string) diag.Field {
	return ecsString("dns.op_code", value)
}

// ID create the ECS complain 'dns.id' field.
// The DNS packet identifier assigned by the program that generated the
// query. The identifier is copied to the response.
func (nsDNS) ID(value string) diag.Field {
	return ecsString("dns.id", value)
}

// ## dns.answers fields

// Class create the ECS complain 'dns.answers.class' field.
// The class of DNS data contained in this resource record.
func (nsDNSAnswers) Class(value string) diag.Field {
	return ecsString("dns.answers.class", value)
}

// Name create the ECS complain 'dns.answers.name' field.
// The domain name to which this resource record pertains. If a chain of
// CNAME is being resolved, each answer's `name` should be the one that
// corresponds with the answer's `data`. It should not simply be the
// original `question.name` repeated.
func (nsDNSAnswers) Name(value string) diag.Field {
	return ecsString("dns.answers.name", value)
}

// Data create the ECS complain 'dns.answers.data' field.
// The data describing the resource. The meaning of this data depends on
// the type and class of the resource record.
func (nsDNSAnswers) Data(value string) diag.Field {
	return ecsString("dns.answers.data", value)
}

// TTL create the ECS complain 'dns.answers.ttl' field.
// The time interval in seconds that this resource record may be cached
// before it should be discarded. Zero values mean that the data should
// not be cached.
func (nsDNSAnswers) TTL(value int64) diag.Field {
	return ecsInt64("dns.answers.ttl", value)
}

// Type create the ECS complain 'dns.answers.type' field.
// The type of data contained in this resource record.
func (nsDNSAnswers) Type(value string) diag.Field {
	return ecsString("dns.answers.type", value)
}

// ## dns.question fields

// RegisteredDomain create the ECS complain 'dns.question.registered_domain' field.
// The highest registered domain, stripped of the subdomain. For example,
// the registered domain for "foo.google.com" is "google.com". This value
// can be determined precisely with a list like the public suffix list
// (http://publicsuffix.org). Trying to approximate this by simply taking
// the last two labels will not work well for TLDs such as "co.uk".
func (nsDNSQuestion) RegisteredDomain(value string) diag.Field {
	return ecsString("dns.question.registered_domain", value)
}

// TopLevelDomain create the ECS complain 'dns.question.top_level_domain' field.
// The effective top level domain (eTLD), also known as the domain suffix,
// is the last part of the domain name. For example, the top level domain
// for google.com is "com". This value can be determined precisely with a
// list like the public suffix list (http://publicsuffix.org). Trying to
// approximate this by simply taking the last label will not work well for
// effective TLDs such as "co.uk".
func (nsDNSQuestion) TopLevelDomain(value string) diag.Field {
	return ecsString("dns.question.top_level_domain", value)
}

// Name create the ECS complain 'dns.question.name' field.
// The name being queried. If the name field contains non-printable
// characters (below 32 or above 126), those characters should be
// represented as escaped base 10 integers (\DDD). Back slashes and quotes
// should be escaped. Tabs, carriage returns, and line feeds should be
// converted to \t, \r, and \n respectively.
func (nsDNSQuestion) Name(value string) diag.Field {
	return ecsString("dns.question.name", value)
}

// Subdomain create the ECS complain 'dns.question.subdomain' field.
// The subdomain is all of the labels under the registered_domain. If the
// domain has multiple levels of subdomain, such as
// "sub2.sub1.example.com", the subdomain field should contain
// "sub2.sub1", with no trailing period.
func (nsDNSQuestion) Subdomain(value string) diag.Field {
	return ecsString("dns.question.subdomain", value)
}

// Class create the ECS complain 'dns.question.class' field.
// The class of records being queried.
func (nsDNSQuestion) Class(value string) diag.Field {
	return ecsString("dns.question.class", value)
}

// Type create the ECS complain 'dns.question.type' field.
// The type of record being queried.
func (nsDNSQuestion) Type(value string) diag.Field {
	return ecsString("dns.question.type", value)
}

// ## ecs fields

// Version create the ECS complain 'ecs.version' field.
// ECS version this event conforms to. `ecs.version` is a required field
// and must exist in all events. When querying across multiple indices --
// which may conform to slightly different ECS versions -- this field lets
// integrations adjust to the schema version of the events.
func (nsECS) Version(value string) diag.Field {
	return ecsString("ecs.version", value)
}

// ## error fields

// Message create the ECS complain 'error.message' field.
// Error message.
func (nsError) Message(value string) diag.Field {
	return ecsString("error.message", value)
}

// Code create the ECS complain 'error.code' field.
// Error code describing the error.
func (nsError) Code(value string) diag.Field {
	return ecsString("error.code", value)
}

// Type create the ECS complain 'error.type' field.
// The type of the error, for example the class name of the exception.
func (nsError) Type(value string) diag.Field {
	return ecsString("error.type", value)
}

// StackTrace create the ECS complain 'error.stack_trace' field.
// The stack trace of this error in plain text.
func (nsError) StackTrace(value string) diag.Field {
	return ecsString("error.stack_trace", value)
}

// ID create the ECS complain 'error.id' field.
// Unique identifier for the error.
func (nsError) ID(value string) diag.Field {
	return ecsString("error.id", value)
}

// ## event fields

// Sequence create the ECS complain 'event.sequence' field.
// Sequence number of the event. The sequence number is a value published
// by some event sources, to make the exact ordering of events
// unambiguous, regarless of the timestamp precision.
func (nsEvent) Sequence(value int64) diag.Field {
	return ecsInt64("event.sequence", value)
}

// Code create the ECS complain 'event.code' field.
// Identification code for this event, if one exists. Some event sources
// use event codes to identify messages unambiguously, regardless of
// message language or wording adjustments over time. An example of this
// is the Windows Event ID.
func (nsEvent) Code(value string) diag.Field {
	return ecsString("event.code", value)
}

// Module create the ECS complain 'event.module' field.
// Name of the module this data is coming from. If your monitoring agent
// supports the concept of modules or plugins to process events of a given
// source (e.g. Apache logs), `event.module` should contain the name of
// this module.
func (nsEvent) Module(value string) diag.Field {
	return ecsString("event.module", value)
}

// ID create the ECS complain 'event.id' field.
// Unique ID to describe the event.
func (nsEvent) ID(value string) diag.Field {
	return ecsString("event.id", value)
}

// Category create the ECS complain 'event.category' field.
// This is one of four ECS Categorization Fields, and indicates the second
// level in the ECS category hierarchy. `event.category` represents the
// "big buckets" of ECS categories. For example, filtering on
// `event.category:process` yields all events relating to process
// activity. This field is closely related to `event.type`, which is used
// as a subcategory. This field is an array. This will allow proper
// categorization of some events that fall in multiple categories.
func (nsEvent) Category(value string) diag.Field {
	return ecsString("event.category", value)
}

// Outcome create the ECS complain 'event.outcome' field.
// This is one of four ECS Categorization Fields, and indicates the lowest
// level in the ECS category hierarchy. `event.outcome` simply denotes
// whether the event represent a success or a failure. Note that not all
// events will have an associated outcome. For example, this field is
// generally not populated for metric events or events with
// `event.type:info`.
func (nsEvent) Outcome(value string) diag.Field {
	return ecsString("event.outcome", value)
}

// Timezone create the ECS complain 'event.timezone' field.
// This field should be populated when the event's timestamp does not
// include timezone information already (e.g. default Syslog timestamps).
// It's optional otherwise. Acceptable timezone formats are: a canonical
// ID (e.g. "Europe/Amsterdam"), abbreviated (e.g. "EST") or an HH:mm
// differential (e.g. "-05:00").
func (nsEvent) Timezone(value string) diag.Field {
	return ecsString("event.timezone", value)
}

// Original create the ECS complain 'event.original' field.
// Raw text message of entire event. Used to demonstrate log integrity.
// This field is not indexed and doc_values are disabled. It cannot be
// searched, but it can be retrieved from `_source`.
func (nsEvent) Original(value string) diag.Field {
	return ecsString("event.original", value)
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
func (nsEvent) Created(value time.Time) diag.Field {
	return ecsTime("event.created", value)
}

// Provider create the ECS complain 'event.provider' field.
// Source of the event. Event transports such as Syslog or the Windows
// Event Log typically mention the source of an event. It can be the name
// of the software that generated the event (e.g. Sysmon, httpd), or of a
// subsystem of the operating system (kernel,
// Microsoft-Windows-Security-Auditing).
func (nsEvent) Provider(value string) diag.Field {
	return ecsString("event.provider", value)
}

// Hash create the ECS complain 'event.hash' field.
// Hash (perhaps logstash fingerprint) of raw field to be able to
// demonstrate log integrity.
func (nsEvent) Hash(value string) diag.Field {
	return ecsString("event.hash", value)
}

// Duration create the ECS complain 'event.duration' field.
// Duration of the event in nanoseconds. If event.start and event.end are
// known this value should be the difference between the end and start
// time.
func (nsEvent) Duration(value int64) diag.Field {
	return ecsInt64("event.duration", value)
}

// End create the ECS complain 'event.end' field.
// event.end contains the date when the event ended or when the activity
// was last observed.
func (nsEvent) End(value time.Time) diag.Field {
	return ecsTime("event.end", value)
}

// Action create the ECS complain 'event.action' field.
// The action captured by the event. This describes the information in the
// event. It is more specific than `event.category`. Examples are
// `group-add`, `process-started`, `file-created`. The value is normally
// defined by the implementer.
func (nsEvent) Action(value string) diag.Field {
	return ecsString("event.action", value)
}

// Start create the ECS complain 'event.start' field.
// event.start contains the date when the event started or when the
// activity was first observed.
func (nsEvent) Start(value time.Time) diag.Field {
	return ecsTime("event.start", value)
}

// RiskScore create the ECS complain 'event.risk_score' field.
// Risk score or priority of the event (e.g. security solutions). Use your
// system's original value here.
func (nsEvent) RiskScore(value float64) diag.Field {
	return ecsFloat64("event.risk_score", value)
}

// Severity create the ECS complain 'event.severity' field.
// The numeric severity of the event according to your event source. What
// the different severity values mean can be different between sources and
// use cases. It's up to the implementer to make sure severities are
// consistent across events from the same source. The Syslog severity
// belongs in `log.syslog.severity.code`. `event.severity` is meant to
// represent the severity according to the event source (e.g. firewall,
// IDS). If the event source does not publish its own severity, you may
// optionally copy the `log.syslog.severity.code` to `event.severity`.
func (nsEvent) Severity(value int64) diag.Field {
	return ecsInt64("event.severity", value)
}

// Dataset create the ECS complain 'event.dataset' field.
// Name of the dataset. If an event source publishes more than one type of
// log or events (e.g. access log, error log), the dataset is used to
// specify which one the event comes from. It's recommended but not
// required to start the dataset name with the module name, followed by a
// dot, then the dataset name.
func (nsEvent) Dataset(value string) diag.Field {
	return ecsString("event.dataset", value)
}

// RiskScoreNorm create the ECS complain 'event.risk_score_norm' field.
// Normalized risk score or priority of the event, on a scale of 0 to 100.
// This is mainly useful if you use more than one system that assigns risk
// scores, and you want to see a normalized value across all systems.
func (nsEvent) RiskScoreNorm(value float64) diag.Field {
	return ecsFloat64("event.risk_score_norm", value)
}

// Kind create the ECS complain 'event.kind' field.
// This is one of four ECS Categorization Fields, and indicates the
// highest level in the ECS category hierarchy. `event.kind` gives
// high-level information about what type of information the event
// contains, without being specific to the contents of the event. For
// example, values of this field distinguish alert events from metric
// events. The value of this field can be used to inform how these kinds
// of events should be handled. They may warrant different retention,
// different access control, it may also help understand whether the data
// coming in at a regular interval or not.
func (nsEvent) Kind(value string) diag.Field {
	return ecsString("event.kind", value)
}

// Ingested create the ECS complain 'event.ingested' field.
// Timestamp when an event arrived in the central data store. This is
// different from `@timestamp`, which is when the event originally
// occurred.  It's also different from `event.created`, which is meant to
// capture the first time an agent saw the event. In normal conditions,
// assuming no tampering, the timestamps should chronologically look like
// this: `@timestamp` < `event.created` < `event.ingested`.
func (nsEvent) Ingested(value time.Time) diag.Field {
	return ecsTime("event.ingested", value)
}

// Type create the ECS complain 'event.type' field.
// This is one of four ECS Categorization Fields, and indicates the third
// level in the ECS category hierarchy. `event.type` represents a
// categorization "sub-bucket" that, when used along with the
// `event.category` field values, enables filtering events down to a level
// appropriate for single visualization. This field is an array. This will
// allow proper categorization of some events that fall in multiple event
// types.
func (nsEvent) Type(value string) diag.Field {
	return ecsString("event.type", value)
}

// ## file fields

// TargetPath create the ECS complain 'file.target_path' field.
// Target path for symlinks.
func (nsFile) TargetPath(value string) diag.Field {
	return ecsString("file.target_path", value)
}

// Accessed create the ECS complain 'file.accessed' field.
// Last time the file was accessed. Note that not all filesystems keep
// track of access time.
func (nsFile) Accessed(value time.Time) diag.Field {
	return ecsTime("file.accessed", value)
}

// UID create the ECS complain 'file.uid' field.
// The user ID (UID) or security identifier (SID) of the file owner.
func (nsFile) UID(value string) diag.Field {
	return ecsString("file.uid", value)
}

// Path create the ECS complain 'file.path' field.
// Full path to the file, including the file name. It should include the
// drive letter, when appropriate.
func (nsFile) Path(value string) diag.Field {
	return ecsString("file.path", value)
}

// Attributes create the ECS complain 'file.attributes' field.
// Array of file attributes. Attributes names will vary by platform.
// Here's a non-exhaustive list of values that are expected in this field:
// archive, compressed, directory, encrypted, execute, hidden, read,
// readonly, system, write.
func (nsFile) Attributes(value string) diag.Field {
	return ecsString("file.attributes", value)
}

// Name create the ECS complain 'file.name' field.
// Name of the file including the extension, without the directory.
func (nsFile) Name(value string) diag.Field {
	return ecsString("file.name", value)
}

// Owner create the ECS complain 'file.owner' field.
// File owner's username.
func (nsFile) Owner(value string) diag.Field {
	return ecsString("file.owner", value)
}

// Mode create the ECS complain 'file.mode' field.
// Mode of the file in octal representation.
func (nsFile) Mode(value string) diag.Field {
	return ecsString("file.mode", value)
}

// Inode create the ECS complain 'file.inode' field.
// Inode representing the file in the filesystem.
func (nsFile) Inode(value string) diag.Field {
	return ecsString("file.inode", value)
}

// Type create the ECS complain 'file.type' field.
// File type (file, dir, or symlink).
func (nsFile) Type(value string) diag.Field {
	return ecsString("file.type", value)
}

// Extension create the ECS complain 'file.extension' field.
// File extension.
func (nsFile) Extension(value string) diag.Field {
	return ecsString("file.extension", value)
}

// Directory create the ECS complain 'file.directory' field.
// Directory where the file is located. It should include the drive
// letter, when appropriate.
func (nsFile) Directory(value string) diag.Field {
	return ecsString("file.directory", value)
}

// Gid create the ECS complain 'file.gid' field.
// Primary group ID (GID) of the file.
func (nsFile) Gid(value string) diag.Field {
	return ecsString("file.gid", value)
}

// Mtime create the ECS complain 'file.mtime' field.
// Last time the file content was modified.
func (nsFile) Mtime(value time.Time) diag.Field {
	return ecsTime("file.mtime", value)
}

// Ctime create the ECS complain 'file.ctime' field.
// Last time the file attributes or metadata changed. Note that changes to
// the file content will update `mtime`. This implies `ctime` will be
// adjusted at the same time, since `mtime` is an attribute of the file.
func (nsFile) Ctime(value time.Time) diag.Field {
	return ecsTime("file.ctime", value)
}

// Device create the ECS complain 'file.device' field.
// Device that is the source of the file.
func (nsFile) Device(value string) diag.Field {
	return ecsString("file.device", value)
}

// DriveLetter create the ECS complain 'file.drive_letter' field.
// Drive letter where the file is located. This field is only relevant on
// Windows. The value should be uppercase, and not include the colon.
func (nsFile) DriveLetter(value string) diag.Field {
	return ecsString("file.drive_letter", value)
}

// Created create the ECS complain 'file.created' field.
// File creation time. Note that not all filesystems store the creation
// time.
func (nsFile) Created(value time.Time) diag.Field {
	return ecsTime("file.created", value)
}

// Group create the ECS complain 'file.group' field.
// Primary group name of the file.
func (nsFile) Group(value string) diag.Field {
	return ecsString("file.group", value)
}

// Size create the ECS complain 'file.size' field.
// File size in bytes. Only relevant when `file.type` is "file".
func (nsFile) Size(value int64) diag.Field {
	return ecsInt64("file.size", value)
}

// ## file.hash fields

// Sha1 create the ECS complain 'file.hash.sha1' field.
// SHA1 hash.
func (nsFileHash) Sha1(value string) diag.Field {
	return ecsString("file.hash.sha1", value)
}

// Sha512 create the ECS complain 'file.hash.sha512' field.
// SHA512 hash.
func (nsFileHash) Sha512(value string) diag.Field {
	return ecsString("file.hash.sha512", value)
}

// Sha256 create the ECS complain 'file.hash.sha256' field.
// SHA256 hash.
func (nsFileHash) Sha256(value string) diag.Field {
	return ecsString("file.hash.sha256", value)
}

// Md5 create the ECS complain 'file.hash.md5' field.
// MD5 hash.
func (nsFileHash) Md5(value string) diag.Field {
	return ecsString("file.hash.md5", value)
}

// ## geo fields

// ContinentName create the ECS complain 'geo.continent_name' field.
// Name of the continent.
func (nsGeo) ContinentName(value string) diag.Field {
	return ecsString("geo.continent_name", value)
}

// Name create the ECS complain 'geo.name' field.
// User-defined description of a location, at the level of granularity
// they care about. Could be the name of their data centers, the floor
// number, if this describes a local physical entity, city names. Not
// typically used in automated geolocation.
func (nsGeo) Name(value string) diag.Field {
	return ecsString("geo.name", value)
}

// CountryName create the ECS complain 'geo.country_name' field.
// Country name.
func (nsGeo) CountryName(value string) diag.Field {
	return ecsString("geo.country_name", value)
}

// RegionName create the ECS complain 'geo.region_name' field.
// Region name.
func (nsGeo) RegionName(value string) diag.Field {
	return ecsString("geo.region_name", value)
}

// CityName create the ECS complain 'geo.city_name' field.
// City name.
func (nsGeo) CityName(value string) diag.Field {
	return ecsString("geo.city_name", value)
}

// CountryIsoCode create the ECS complain 'geo.country_iso_code' field.
// Country ISO code.
func (nsGeo) CountryIsoCode(value string) diag.Field {
	return ecsString("geo.country_iso_code", value)
}

// RegionIsoCode create the ECS complain 'geo.region_iso_code' field.
// Region ISO code.
func (nsGeo) RegionIsoCode(value string) diag.Field {
	return ecsString("geo.region_iso_code", value)
}

// Location create the ECS complain 'geo.location' field.
// Longitude and latitude.
func (nsGeo) Location(value string) diag.Field {
	return ecsString("geo.location", value)
}

// ## group fields

// ID create the ECS complain 'group.id' field.
// Unique identifier for the group on the system/platform.
func (nsGroup) ID(value string) diag.Field {
	return ecsString("group.id", value)
}

// Domain create the ECS complain 'group.domain' field.
// Name of the directory the group is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsGroup) Domain(value string) diag.Field {
	return ecsString("group.domain", value)
}

// Name create the ECS complain 'group.name' field.
// Name of the group.
func (nsGroup) Name(value string) diag.Field {
	return ecsString("group.name", value)
}

// ## hash fields

// Sha1 create the ECS complain 'hash.sha1' field.
// SHA1 hash.
func (nsHash) Sha1(value string) diag.Field {
	return ecsString("hash.sha1", value)
}

// Md5 create the ECS complain 'hash.md5' field.
// MD5 hash.
func (nsHash) Md5(value string) diag.Field {
	return ecsString("hash.md5", value)
}

// Sha512 create the ECS complain 'hash.sha512' field.
// SHA512 hash.
func (nsHash) Sha512(value string) diag.Field {
	return ecsString("hash.sha512", value)
}

// Sha256 create the ECS complain 'hash.sha256' field.
// SHA256 hash.
func (nsHash) Sha256(value string) diag.Field {
	return ecsString("hash.sha256", value)
}

// ## host fields

// MAC create the ECS complain 'host.mac' field.
// Host mac address.
func (nsHost) MAC(value string) diag.Field {
	return ecsString("host.mac", value)
}

// Hostname create the ECS complain 'host.hostname' field.
// Hostname of the host. It normally contains what the `hostname` command
// returns on the host machine.
func (nsHost) Hostname(value string) diag.Field {
	return ecsString("host.hostname", value)
}

// IP create the ECS complain 'host.ip' field.
// Host ip address.
func (nsHost) IP(value string) diag.Field {
	return ecsString("host.ip", value)
}

// Architecture create the ECS complain 'host.architecture' field.
// Operating system architecture.
func (nsHost) Architecture(value string) diag.Field {
	return ecsString("host.architecture", value)
}

// Name create the ECS complain 'host.name' field.
// Name of the host. It can contain what `hostname` returns on Unix
// systems, the fully qualified domain name, or a name specified by the
// user. The sender decides which value to use.
func (nsHost) Name(value string) diag.Field {
	return ecsString("host.name", value)
}

// Uptime create the ECS complain 'host.uptime' field.
// Seconds the host has been up.
func (nsHost) Uptime(value int64) diag.Field {
	return ecsInt64("host.uptime", value)
}

// ID create the ECS complain 'host.id' field.
// Unique host id. As hostname is not always unique, use values that are
// meaningful in your environment. Example: The current usage of
// `beat.name`.
func (nsHost) ID(value string) diag.Field {
	return ecsString("host.id", value)
}

// Domain create the ECS complain 'host.domain' field.
// Name of the domain of which the host is a member.  For example, on
// Windows this could be the host's Active Directory domain or NetBIOS
// domain name.  For Linux this could be the domain of the host's LDAP
// provider.
func (nsHost) Domain(value string) diag.Field {
	return ecsString("host.domain", value)
}

// Type create the ECS complain 'host.type' field.
// Type of host. For Cloud providers this can be the machine type like
// `t2.medium`. If vm, this could be the container, for example, or other
// information meaningful in your environment.
func (nsHost) Type(value string) diag.Field {
	return ecsString("host.type", value)
}

// ## host.geo fields

// CityName create the ECS complain 'host.geo.city_name' field.
// City name.
func (nsHostGeo) CityName(value string) diag.Field {
	return ecsString("host.geo.city_name", value)
}

// CountryName create the ECS complain 'host.geo.country_name' field.
// Country name.
func (nsHostGeo) CountryName(value string) diag.Field {
	return ecsString("host.geo.country_name", value)
}

// RegionIsoCode create the ECS complain 'host.geo.region_iso_code' field.
// Region ISO code.
func (nsHostGeo) RegionIsoCode(value string) diag.Field {
	return ecsString("host.geo.region_iso_code", value)
}

// Name create the ECS complain 'host.geo.name' field.
// User-defined description of a location, at the level of granularity
// they care about. Could be the name of their data centers, the floor
// number, if this describes a local physical entity, city names. Not
// typically used in automated geolocation.
func (nsHostGeo) Name(value string) diag.Field {
	return ecsString("host.geo.name", value)
}

// CountryIsoCode create the ECS complain 'host.geo.country_iso_code' field.
// Country ISO code.
func (nsHostGeo) CountryIsoCode(value string) diag.Field {
	return ecsString("host.geo.country_iso_code", value)
}

// Location create the ECS complain 'host.geo.location' field.
// Longitude and latitude.
func (nsHostGeo) Location(value string) diag.Field {
	return ecsString("host.geo.location", value)
}

// RegionName create the ECS complain 'host.geo.region_name' field.
// Region name.
func (nsHostGeo) RegionName(value string) diag.Field {
	return ecsString("host.geo.region_name", value)
}

// ContinentName create the ECS complain 'host.geo.continent_name' field.
// Name of the continent.
func (nsHostGeo) ContinentName(value string) diag.Field {
	return ecsString("host.geo.continent_name", value)
}

// ## host.os fields

// Platform create the ECS complain 'host.os.platform' field.
// Operating system platform (such centos, ubuntu, windows).
func (nsHostOS) Platform(value string) diag.Field {
	return ecsString("host.os.platform", value)
}

// Family create the ECS complain 'host.os.family' field.
// OS family (such as redhat, debian, freebsd, windows).
func (nsHostOS) Family(value string) diag.Field {
	return ecsString("host.os.family", value)
}

// Kernel create the ECS complain 'host.os.kernel' field.
// Operating system kernel version as a raw string.
func (nsHostOS) Kernel(value string) diag.Field {
	return ecsString("host.os.kernel", value)
}

// Full create the ECS complain 'host.os.full' field.
// Operating system name, including the version or code name.
func (nsHostOS) Full(value string) diag.Field {
	return ecsString("host.os.full", value)
}

// Version create the ECS complain 'host.os.version' field.
// Operating system version as a raw string.
func (nsHostOS) Version(value string) diag.Field {
	return ecsString("host.os.version", value)
}

// Name create the ECS complain 'host.os.name' field.
// Operating system name, without the version.
func (nsHostOS) Name(value string) diag.Field {
	return ecsString("host.os.name", value)
}

// ## host.user fields

// Hash create the ECS complain 'host.user.hash' field.
// Unique user hash to correlate information for a user in anonymized
// form. Useful if `user.id` or `user.name` contain confidential
// information and cannot be used.
func (nsHostUser) Hash(value string) diag.Field {
	return ecsString("host.user.hash", value)
}

// Name create the ECS complain 'host.user.name' field.
// Short name or login of the user.
func (nsHostUser) Name(value string) diag.Field {
	return ecsString("host.user.name", value)
}

// FullName create the ECS complain 'host.user.full_name' field.
// User's full name, if available.
func (nsHostUser) FullName(value string) diag.Field {
	return ecsString("host.user.full_name", value)
}

// Domain create the ECS complain 'host.user.domain' field.
// Name of the directory the user is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsHostUser) Domain(value string) diag.Field {
	return ecsString("host.user.domain", value)
}

// ID create the ECS complain 'host.user.id' field.
// One or multiple unique identifiers of the user.
func (nsHostUser) ID(value string) diag.Field {
	return ecsString("host.user.id", value)
}

// Email create the ECS complain 'host.user.email' field.
// User email address.
func (nsHostUser) Email(value string) diag.Field {
	return ecsString("host.user.email", value)
}

// ## host.user.group fields

// Name create the ECS complain 'host.user.group.name' field.
// Name of the group.
func (nsHostUserGroup) Name(value string) diag.Field {
	return ecsString("host.user.group.name", value)
}

// ID create the ECS complain 'host.user.group.id' field.
// Unique identifier for the group on the system/platform.
func (nsHostUserGroup) ID(value string) diag.Field {
	return ecsString("host.user.group.id", value)
}

// Domain create the ECS complain 'host.user.group.domain' field.
// Name of the directory the group is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsHostUserGroup) Domain(value string) diag.Field {
	return ecsString("host.user.group.domain", value)
}

// ## http fields

// Version create the ECS complain 'http.version' field.
// HTTP version.
func (nsHTTP) Version(value string) diag.Field {
	return ecsString("http.version", value)
}

// ## http.request fields

// Bytes create the ECS complain 'http.request.bytes' field.
// Total size in bytes of the request (body and headers).
func (nsHTTPRequest) Bytes(value int64) diag.Field {
	return ecsInt64("http.request.bytes", value)
}

// Referrer create the ECS complain 'http.request.referrer' field.
// Referrer for this HTTP request.
func (nsHTTPRequest) Referrer(value string) diag.Field {
	return ecsString("http.request.referrer", value)
}

// Method create the ECS complain 'http.request.method' field.
// HTTP request method. The field value must be normalized to lowercase
// for querying. See the documentation section "Implementing ECS".
func (nsHTTPRequest) Method(value string) diag.Field {
	return ecsString("http.request.method", value)
}

// ## http.request.body fields

// Content create the ECS complain 'http.request.body.content' field.
// The full HTTP request body.
func (nsHTTPRequestBody) Content(value string) diag.Field {
	return ecsString("http.request.body.content", value)
}

// Bytes create the ECS complain 'http.request.body.bytes' field.
// Size in bytes of the request body.
func (nsHTTPRequestBody) Bytes(value int64) diag.Field {
	return ecsInt64("http.request.body.bytes", value)
}

// ## http.response fields

// StatusCode create the ECS complain 'http.response.status_code' field.
// HTTP response status code.
func (nsHTTPResponse) StatusCode(value int64) diag.Field {
	return ecsInt64("http.response.status_code", value)
}

// Bytes create the ECS complain 'http.response.bytes' field.
// Total size in bytes of the response (body and headers).
func (nsHTTPResponse) Bytes(value int64) diag.Field {
	return ecsInt64("http.response.bytes", value)
}

// ## http.response.body fields

// Content create the ECS complain 'http.response.body.content' field.
// The full HTTP response body.
func (nsHTTPResponseBody) Content(value string) diag.Field {
	return ecsString("http.response.body.content", value)
}

// Bytes create the ECS complain 'http.response.body.bytes' field.
// Size in bytes of the response body.
func (nsHTTPResponseBody) Bytes(value int64) diag.Field {
	return ecsInt64("http.response.body.bytes", value)
}

// ## log fields

// Logger create the ECS complain 'log.logger' field.
// The name of the logger inside an application. This is usually the name
// of the class which initialized the logger, or can be a custom name.
func (nsLog) Logger(value string) diag.Field {
	return ecsString("log.logger", value)
}

// Original create the ECS complain 'log.original' field.
// This is the original log message and contains the full log message
// before splitting it up in multiple parts. In contrast to the `message`
// field which can contain an extracted part of the log message, this
// field contains the original, full log message. It can have already some
// modifications applied like encoding or new lines removed to clean up
// the log message. This field is not indexed and doc_values are disabled
// so it can't be queried but the value can be retrieved from `_source`.
func (nsLog) Original(value string) diag.Field {
	return ecsString("log.original", value)
}

// Level create the ECS complain 'log.level' field.
// Original log level of the log event. If the source of the event
// provides a log level or textual severity, this is the one that goes in
// `log.level`. If your source doesn't specify one, you may put your event
// transport's severity here (e.g. Syslog severity). Some examples are
// `warn`, `err`, `i`, `informational`.
func (nsLog) Level(value string) diag.Field {
	return ecsString("log.level", value)
}

// ## log.origin fields

// Function create the ECS complain 'log.origin.function' field.
// The name of the function or method which originated the log event.
func (nsLogOrigin) Function(value string) diag.Field {
	return ecsString("log.origin.function", value)
}

// ## log.origin.file fields

// Line create the ECS complain 'log.origin.file.line' field.
// The line number of the file containing the source code which originated
// the log event.
func (nsLogOriginFile) Line(value int) diag.Field {
	return ecsInt("log.origin.file.line", value)
}

// Name create the ECS complain 'log.origin.file.name' field.
// The name of the file containing the source code which originated the
// log event. Note that this is not the name of the log file.
func (nsLogOriginFile) Name(value string) diag.Field {
	return ecsString("log.origin.file.name", value)
}

// ## log.syslog fields

// Priority create the ECS complain 'log.syslog.priority' field.
// Syslog numeric priority of the event, if available. According to RFCs
// 5424 and 3164, the priority is 8 * facility + severity. This number is
// therefore expected to contain a value between 0 and 191.
func (nsLogSyslog) Priority(value int64) diag.Field {
	return ecsInt64("log.syslog.priority", value)
}

// ## log.syslog.facility fields

// Code create the ECS complain 'log.syslog.facility.code' field.
// The Syslog numeric facility of the log event, if available. According
// to RFCs 5424 and 3164, this value should be an integer between 0 and
// 23.
func (nsLogSyslogFacility) Code(value int64) diag.Field {
	return ecsInt64("log.syslog.facility.code", value)
}

// Name create the ECS complain 'log.syslog.facility.name' field.
// The Syslog text-based facility of the log event, if available.
func (nsLogSyslogFacility) Name(value string) diag.Field {
	return ecsString("log.syslog.facility.name", value)
}

// ## log.syslog.severity fields

// Code create the ECS complain 'log.syslog.severity.code' field.
// The Syslog numeric severity of the log event, if available. If the
// event source publishing via Syslog provides a different numeric
// severity value (e.g. firewall, IDS), your source's numeric severity
// should go to `event.severity`. If the event source does not specify a
// distinct severity, you can optionally copy the Syslog severity to
// `event.severity`.
func (nsLogSyslogSeverity) Code(value int64) diag.Field {
	return ecsInt64("log.syslog.severity.code", value)
}

// Name create the ECS complain 'log.syslog.severity.name' field.
// The Syslog numeric severity of the log event, if available. If the
// event source publishing via Syslog provides a different severity value
// (e.g. firewall, IDS), your source's text severity should go to
// `log.level`. If the event source does not specify a distinct severity,
// you can optionally copy the Syslog severity to `log.level`.
func (nsLogSyslogSeverity) Name(value string) diag.Field {
	return ecsString("log.syslog.severity.name", value)
}

// ## network fields

// Transport create the ECS complain 'network.transport' field.
// Same as network.iana_number, but instead using the Keyword name of the
// transport layer (udp, tcp, ipv6-icmp, etc.) The field value must be
// normalized to lowercase for querying. See the documentation section
// "Implementing ECS".
func (nsNetwork) Transport(value string) diag.Field {
	return ecsString("network.transport", value)
}

// CommunityID create the ECS complain 'network.community_id' field.
// A hash of source and destination IPs and ports, as well as the protocol
// used in a communication. This is a tool-agnostic standard to identify
// flows. Learn more at https://github.com/corelight/community-id-spec.
func (nsNetwork) CommunityID(value string) diag.Field {
	return ecsString("network.community_id", value)
}

// Name create the ECS complain 'network.name' field.
// Name given by operators to sections of their network.
func (nsNetwork) Name(value string) diag.Field {
	return ecsString("network.name", value)
}

// Packets create the ECS complain 'network.packets' field.
// Total packets transferred in both directions. If `source.packets` and
// `destination.packets` are known, `network.packets` is their sum.
func (nsNetwork) Packets(value int64) diag.Field {
	return ecsInt64("network.packets", value)
}

// Type create the ECS complain 'network.type' field.
// In the OSI Model this would be the Network Layer. ipv4, ipv6, ipsec,
// pim, etc The field value must be normalized to lowercase for querying.
// See the documentation section "Implementing ECS".
func (nsNetwork) Type(value string) diag.Field {
	return ecsString("network.type", value)
}

// ForwardedIP create the ECS complain 'network.forwarded_ip' field.
// Host IP address when the source IP address is the proxy.
func (nsNetwork) ForwardedIP(value string) diag.Field {
	return ecsString("network.forwarded_ip", value)
}

// Direction create the ECS complain 'network.direction' field.
// Direction of the network traffic. Recommended values are:   * inbound
// * outbound   * internal   * external   * unknown  When mapping events
// from a host-based monitoring context, populate this field from the
// host's point of view. When mapping events from a network or
// perimeter-based monitoring context, populate this field from the point
// of view of your network perimeter.
func (nsNetwork) Direction(value string) diag.Field {
	return ecsString("network.direction", value)
}

// IANANumber create the ECS complain 'network.iana_number' field.
// IANA Protocol Number
// (https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml).
// Standardized list of protocols. This aligns well with NetFlow and sFlow
// related logs which use the IANA Protocol Number.
func (nsNetwork) IANANumber(value string) diag.Field {
	return ecsString("network.iana_number", value)
}

// Protocol create the ECS complain 'network.protocol' field.
// L7 Network protocol name. ex. http, lumberjack, transport protocol. The
// field value must be normalized to lowercase for querying. See the
// documentation section "Implementing ECS".
func (nsNetwork) Protocol(value string) diag.Field {
	return ecsString("network.protocol", value)
}

// Application create the ECS complain 'network.application' field.
// A name given to an application level protocol. This can be arbitrarily
// assigned for things like microservices, but also apply to things like
// skype, icq, facebook, twitter. This would be used in situations where
// the vendor or service can be decoded such as from the source/dest IP
// owners, ports, or wire format. The field value must be normalized to
// lowercase for querying. See the documentation section "Implementing
// ECS".
func (nsNetwork) Application(value string) diag.Field {
	return ecsString("network.application", value)
}

// Bytes create the ECS complain 'network.bytes' field.
// Total bytes transferred in both directions. If `source.bytes` and
// `destination.bytes` are known, `network.bytes` is their sum.
func (nsNetwork) Bytes(value int64) diag.Field {
	return ecsInt64("network.bytes", value)
}

// ## observer fields

// Type create the ECS complain 'observer.type' field.
// The type of the observer the data is coming from. There is no
// predefined list of observer types. Some examples are `forwarder`,
// `firewall`, `ids`, `ips`, `proxy`, `poller`, `sensor`, `APM server`.
func (nsObserver) Type(value string) diag.Field {
	return ecsString("observer.type", value)
}

// Vendor create the ECS complain 'observer.vendor' field.
// Vendor name of the observer.
func (nsObserver) Vendor(value string) diag.Field {
	return ecsString("observer.vendor", value)
}

// Name create the ECS complain 'observer.name' field.
// Custom name of the observer. This is a name that can be given to an
// observer. This can be helpful for example if multiple firewalls of the
// same model are used in an organization. If no custom name is needed,
// the field can be left empty.
func (nsObserver) Name(value string) diag.Field {
	return ecsString("observer.name", value)
}

// IP create the ECS complain 'observer.ip' field.
// IP address of the observer.
func (nsObserver) IP(value string) diag.Field {
	return ecsString("observer.ip", value)
}

// SerialNumber create the ECS complain 'observer.serial_number' field.
// Observer serial number.
func (nsObserver) SerialNumber(value string) diag.Field {
	return ecsString("observer.serial_number", value)
}

// Hostname create the ECS complain 'observer.hostname' field.
// Hostname of the observer.
func (nsObserver) Hostname(value string) diag.Field {
	return ecsString("observer.hostname", value)
}

// Version create the ECS complain 'observer.version' field.
// Observer version.
func (nsObserver) Version(value string) diag.Field {
	return ecsString("observer.version", value)
}

// MAC create the ECS complain 'observer.mac' field.
// MAC address of the observer
func (nsObserver) MAC(value string) diag.Field {
	return ecsString("observer.mac", value)
}

// Product create the ECS complain 'observer.product' field.
// The product name of the observer.
func (nsObserver) Product(value string) diag.Field {
	return ecsString("observer.product", value)
}

// ## observer.geo fields

// ContinentName create the ECS complain 'observer.geo.continent_name' field.
// Name of the continent.
func (nsObserverGeo) ContinentName(value string) diag.Field {
	return ecsString("observer.geo.continent_name", value)
}

// Name create the ECS complain 'observer.geo.name' field.
// User-defined description of a location, at the level of granularity
// they care about. Could be the name of their data centers, the floor
// number, if this describes a local physical entity, city names. Not
// typically used in automated geolocation.
func (nsObserverGeo) Name(value string) diag.Field {
	return ecsString("observer.geo.name", value)
}

// Location create the ECS complain 'observer.geo.location' field.
// Longitude and latitude.
func (nsObserverGeo) Location(value string) diag.Field {
	return ecsString("observer.geo.location", value)
}

// RegionIsoCode create the ECS complain 'observer.geo.region_iso_code' field.
// Region ISO code.
func (nsObserverGeo) RegionIsoCode(value string) diag.Field {
	return ecsString("observer.geo.region_iso_code", value)
}

// CountryName create the ECS complain 'observer.geo.country_name' field.
// Country name.
func (nsObserverGeo) CountryName(value string) diag.Field {
	return ecsString("observer.geo.country_name", value)
}

// RegionName create the ECS complain 'observer.geo.region_name' field.
// Region name.
func (nsObserverGeo) RegionName(value string) diag.Field {
	return ecsString("observer.geo.region_name", value)
}

// CityName create the ECS complain 'observer.geo.city_name' field.
// City name.
func (nsObserverGeo) CityName(value string) diag.Field {
	return ecsString("observer.geo.city_name", value)
}

// CountryIsoCode create the ECS complain 'observer.geo.country_iso_code' field.
// Country ISO code.
func (nsObserverGeo) CountryIsoCode(value string) diag.Field {
	return ecsString("observer.geo.country_iso_code", value)
}

// ## observer.os fields

// Platform create the ECS complain 'observer.os.platform' field.
// Operating system platform (such centos, ubuntu, windows).
func (nsObserverOS) Platform(value string) diag.Field {
	return ecsString("observer.os.platform", value)
}

// Kernel create the ECS complain 'observer.os.kernel' field.
// Operating system kernel version as a raw string.
func (nsObserverOS) Kernel(value string) diag.Field {
	return ecsString("observer.os.kernel", value)
}

// Version create the ECS complain 'observer.os.version' field.
// Operating system version as a raw string.
func (nsObserverOS) Version(value string) diag.Field {
	return ecsString("observer.os.version", value)
}

// Name create the ECS complain 'observer.os.name' field.
// Operating system name, without the version.
func (nsObserverOS) Name(value string) diag.Field {
	return ecsString("observer.os.name", value)
}

// Family create the ECS complain 'observer.os.family' field.
// OS family (such as redhat, debian, freebsd, windows).
func (nsObserverOS) Family(value string) diag.Field {
	return ecsString("observer.os.family", value)
}

// Full create the ECS complain 'observer.os.full' field.
// Operating system name, including the version or code name.
func (nsObserverOS) Full(value string) diag.Field {
	return ecsString("observer.os.full", value)
}

// ## organization fields

// ID create the ECS complain 'organization.id' field.
// Unique identifier for the organization.
func (nsOrganization) ID(value string) diag.Field {
	return ecsString("organization.id", value)
}

// Name create the ECS complain 'organization.name' field.
// Organization name.
func (nsOrganization) Name(value string) diag.Field {
	return ecsString("organization.name", value)
}

// ## os fields

// Version create the ECS complain 'os.version' field.
// Operating system version as a raw string.
func (nsOS) Version(value string) diag.Field {
	return ecsString("os.version", value)
}

// Kernel create the ECS complain 'os.kernel' field.
// Operating system kernel version as a raw string.
func (nsOS) Kernel(value string) diag.Field {
	return ecsString("os.kernel", value)
}

// Full create the ECS complain 'os.full' field.
// Operating system name, including the version or code name.
func (nsOS) Full(value string) diag.Field {
	return ecsString("os.full", value)
}

// Family create the ECS complain 'os.family' field.
// OS family (such as redhat, debian, freebsd, windows).
func (nsOS) Family(value string) diag.Field {
	return ecsString("os.family", value)
}

// Platform create the ECS complain 'os.platform' field.
// Operating system platform (such centos, ubuntu, windows).
func (nsOS) Platform(value string) diag.Field {
	return ecsString("os.platform", value)
}

// Name create the ECS complain 'os.name' field.
// Operating system name, without the version.
func (nsOS) Name(value string) diag.Field {
	return ecsString("os.name", value)
}

// ## package fields

// License create the ECS complain 'package.license' field.
// License under which the package was released. Use a short name, e.g.
// the license identifier from SPDX License List where possible
// (https://spdx.org/licenses/).
func (nsPackage) License(value string) diag.Field {
	return ecsString("package.license", value)
}

// Version create the ECS complain 'package.version' field.
// Package version
func (nsPackage) Version(value string) diag.Field {
	return ecsString("package.version", value)
}

// Path create the ECS complain 'package.path' field.
// Path where the package is installed.
func (nsPackage) Path(value string) diag.Field {
	return ecsString("package.path", value)
}

// Description create the ECS complain 'package.description' field.
// Description of the package.
func (nsPackage) Description(value string) diag.Field {
	return ecsString("package.description", value)
}

// BuildVersion create the ECS complain 'package.build_version' field.
// Additional information about the build version of the installed
// package. For example use the commit SHA of a non-released package.
func (nsPackage) BuildVersion(value string) diag.Field {
	return ecsString("package.build_version", value)
}

// InstallScope create the ECS complain 'package.install_scope' field.
// Indicating how the package was installed, e.g. user-local, global.
func (nsPackage) InstallScope(value string) diag.Field {
	return ecsString("package.install_scope", value)
}

// Name create the ECS complain 'package.name' field.
// Package name
func (nsPackage) Name(value string) diag.Field {
	return ecsString("package.name", value)
}

// Checksum create the ECS complain 'package.checksum' field.
// Checksum of the installed package for verification.
func (nsPackage) Checksum(value string) diag.Field {
	return ecsString("package.checksum", value)
}

// Type create the ECS complain 'package.type' field.
// Type of package. This should contain the package file type, rather than
// the package manager name. Examples: rpm, dpkg, brew, npm, gem, nupkg,
// jar.
func (nsPackage) Type(value string) diag.Field {
	return ecsString("package.type", value)
}

// Architecture create the ECS complain 'package.architecture' field.
// Package architecture.
func (nsPackage) Architecture(value string) diag.Field {
	return ecsString("package.architecture", value)
}

// Size create the ECS complain 'package.size' field.
// Package size in bytes.
func (nsPackage) Size(value int64) diag.Field {
	return ecsInt64("package.size", value)
}

// Reference create the ECS complain 'package.reference' field.
// Home page or reference URL of the software in this package, if
// available.
func (nsPackage) Reference(value string) diag.Field {
	return ecsString("package.reference", value)
}

// Installed create the ECS complain 'package.installed' field.
// Time when package was installed.
func (nsPackage) Installed(value time.Time) diag.Field {
	return ecsTime("package.installed", value)
}

// ## process fields

// Pgid create the ECS complain 'process.pgid' field.
// Identifier of the group of processes the process belongs to.
func (nsProcess) Pgid(value int64) diag.Field {
	return ecsInt64("process.pgid", value)
}

// Uptime create the ECS complain 'process.uptime' field.
// Seconds the process has been up.
func (nsProcess) Uptime(value int64) diag.Field {
	return ecsInt64("process.uptime", value)
}

// CommandLine create the ECS complain 'process.command_line' field.
// Full command line that started the process, including the absolute path
// to the executable, and all arguments. Some arguments may be filtered to
// protect sensitive information.
func (nsProcess) CommandLine(value string) diag.Field {
	return ecsString("process.command_line", value)
}

// PPID create the ECS complain 'process.ppid' field.
// Parent process' pid.
func (nsProcess) PPID(value int64) diag.Field {
	return ecsInt64("process.ppid", value)
}

// WorkingDirectory create the ECS complain 'process.working_directory' field.
// The working directory of the process.
func (nsProcess) WorkingDirectory(value string) diag.Field {
	return ecsString("process.working_directory", value)
}

// Name create the ECS complain 'process.name' field.
// Process name. Sometimes called program name or similar.
func (nsProcess) Name(value string) diag.Field {
	return ecsString("process.name", value)
}

// ArgsCount create the ECS complain 'process.args_count' field.
// Length of the process.args array. This field can be useful for querying
// or performing bucket analysis on how many arguments were provided to
// start a process. More arguments may be an indication of suspicious
// activity.
func (nsProcess) ArgsCount(value int64) diag.Field {
	return ecsInt64("process.args_count", value)
}

// Start create the ECS complain 'process.start' field.
// The time the process started.
func (nsProcess) Start(value time.Time) diag.Field {
	return ecsTime("process.start", value)
}

// PID create the ECS complain 'process.pid' field.
// Process id.
func (nsProcess) PID(value int64) diag.Field {
	return ecsInt64("process.pid", value)
}

// Args create the ECS complain 'process.args' field.
// Array of process arguments, starting with the absolute path to the
// executable. May be filtered to protect sensitive information.
func (nsProcess) Args(value string) diag.Field {
	return ecsString("process.args", value)
}

// ExitCode create the ECS complain 'process.exit_code' field.
// The exit code of the process, if this is a termination event. The field
// should be absent if there is no exit code for the event (e.g. process
// start).
func (nsProcess) ExitCode(value int64) diag.Field {
	return ecsInt64("process.exit_code", value)
}

// Title create the ECS complain 'process.title' field.
// Process title. The proctitle, some times the same as process name. Can
// also be different: for example a browser setting its title to the web
// page currently opened.
func (nsProcess) Title(value string) diag.Field {
	return ecsString("process.title", value)
}

// Executable create the ECS complain 'process.executable' field.
// Absolute path to the process executable.
func (nsProcess) Executable(value string) diag.Field {
	return ecsString("process.executable", value)
}

// ## process.hash fields

// Md5 create the ECS complain 'process.hash.md5' field.
// MD5 hash.
func (nsProcessHash) Md5(value string) diag.Field {
	return ecsString("process.hash.md5", value)
}

// Sha1 create the ECS complain 'process.hash.sha1' field.
// SHA1 hash.
func (nsProcessHash) Sha1(value string) diag.Field {
	return ecsString("process.hash.sha1", value)
}

// Sha512 create the ECS complain 'process.hash.sha512' field.
// SHA512 hash.
func (nsProcessHash) Sha512(value string) diag.Field {
	return ecsString("process.hash.sha512", value)
}

// Sha256 create the ECS complain 'process.hash.sha256' field.
// SHA256 hash.
func (nsProcessHash) Sha256(value string) diag.Field {
	return ecsString("process.hash.sha256", value)
}

// ## process.parent fields

// PPID create the ECS complain 'process.parent.ppid' field.
// Parent process' pid.
func (nsProcessParent) PPID(value int64) diag.Field {
	return ecsInt64("process.parent.ppid", value)
}

// Start create the ECS complain 'process.parent.start' field.
// The time the process started.
func (nsProcessParent) Start(value time.Time) diag.Field {
	return ecsTime("process.parent.start", value)
}

// ExitCode create the ECS complain 'process.parent.exit_code' field.
// The exit code of the process, if this is a termination event. The field
// should be absent if there is no exit code for the event (e.g. process
// start).
func (nsProcessParent) ExitCode(value int64) diag.Field {
	return ecsInt64("process.parent.exit_code", value)
}

// PID create the ECS complain 'process.parent.pid' field.
// Process id.
func (nsProcessParent) PID(value int64) diag.Field {
	return ecsInt64("process.parent.pid", value)
}

// Executable create the ECS complain 'process.parent.executable' field.
// Absolute path to the process executable.
func (nsProcessParent) Executable(value string) diag.Field {
	return ecsString("process.parent.executable", value)
}

// Uptime create the ECS complain 'process.parent.uptime' field.
// Seconds the process has been up.
func (nsProcessParent) Uptime(value int64) diag.Field {
	return ecsInt64("process.parent.uptime", value)
}

// WorkingDirectory create the ECS complain 'process.parent.working_directory' field.
// The working directory of the process.
func (nsProcessParent) WorkingDirectory(value string) diag.Field {
	return ecsString("process.parent.working_directory", value)
}

// Title create the ECS complain 'process.parent.title' field.
// Process title. The proctitle, some times the same as process name. Can
// also be different: for example a browser setting its title to the web
// page currently opened.
func (nsProcessParent) Title(value string) diag.Field {
	return ecsString("process.parent.title", value)
}

// Args create the ECS complain 'process.parent.args' field.
// Array of process arguments. May be filtered to protect sensitive
// information.
func (nsProcessParent) Args(value string) diag.Field {
	return ecsString("process.parent.args", value)
}

// Pgid create the ECS complain 'process.parent.pgid' field.
// Identifier of the group of processes the process belongs to.
func (nsProcessParent) Pgid(value int64) diag.Field {
	return ecsInt64("process.parent.pgid", value)
}

// CommandLine create the ECS complain 'process.parent.command_line' field.
// Full command line that started the process, including the absolute path
// to the executable, and all arguments. Some arguments may be filtered to
// protect sensitive information.
func (nsProcessParent) CommandLine(value string) diag.Field {
	return ecsString("process.parent.command_line", value)
}

// Name create the ECS complain 'process.parent.name' field.
// Process name. Sometimes called program name or similar.
func (nsProcessParent) Name(value string) diag.Field {
	return ecsString("process.parent.name", value)
}

// ArgsCount create the ECS complain 'process.parent.args_count' field.
// Length of the process.args array. This field can be useful for querying
// or performing bucket analysis on how many arguments were provided to
// start a process. More arguments may be an indication of suspicious
// activity.
func (nsProcessParent) ArgsCount(value int64) diag.Field {
	return ecsInt64("process.parent.args_count", value)
}

// ## process.parent.thread fields

// Name create the ECS complain 'process.parent.thread.name' field.
// Thread name.
func (nsProcessParentThread) Name(value string) diag.Field {
	return ecsString("process.parent.thread.name", value)
}

// ID create the ECS complain 'process.parent.thread.id' field.
// Thread ID.
func (nsProcessParentThread) ID(value int64) diag.Field {
	return ecsInt64("process.parent.thread.id", value)
}

// ## process.thread fields

// ID create the ECS complain 'process.thread.id' field.
// Thread ID.
func (nsProcessThread) ID(value int64) diag.Field {
	return ecsInt64("process.thread.id", value)
}

// Name create the ECS complain 'process.thread.name' field.
// Thread name.
func (nsProcessThread) Name(value string) diag.Field {
	return ecsString("process.thread.name", value)
}

// ## registry fields

// Hive create the ECS complain 'registry.hive' field.
// Abbreviated name for the hive.
func (nsRegistry) Hive(value string) diag.Field {
	return ecsString("registry.hive", value)
}

// Value create the ECS complain 'registry.value' field.
// Name of the value written.
func (nsRegistry) Value(value string) diag.Field {
	return ecsString("registry.value", value)
}

// Key create the ECS complain 'registry.key' field.
// Hive-relative path of keys.
func (nsRegistry) Key(value string) diag.Field {
	return ecsString("registry.key", value)
}

// Path create the ECS complain 'registry.path' field.
// Full path, including hive, key and value
func (nsRegistry) Path(value string) diag.Field {
	return ecsString("registry.path", value)
}

// ## registry.data fields

// Type create the ECS complain 'registry.data.type' field.
// Standard registry type for encoding contents
func (nsRegistryData) Type(value string) diag.Field {
	return ecsString("registry.data.type", value)
}

// Bytes create the ECS complain 'registry.data.bytes' field.
// Original bytes written with base64 encoding. For Windows registry
// operations, such as SetValueEx and RegQueryValueEx, this corresponds to
// the data pointed by `lp_data`. This is optional but provides better
// recoverability and should be populated for REG_BINARY encoded values.
func (nsRegistryData) Bytes(value string) diag.Field {
	return ecsString("registry.data.bytes", value)
}

// Strings create the ECS complain 'registry.data.strings' field.
// Content when writing string types. Populated as an array when writing
// string data to the registry. For single string registry types (REG_SZ,
// REG_EXPAND_SZ), this should be an array with one string. For sequences
// of string with REG_MULTI_SZ, this array will be variable length. For
// numeric data, such as REG_DWORD and REG_QWORD, this should be populated
// with the decimal representation (e.g `"1"`).
func (nsRegistryData) Strings(value string) diag.Field {
	return ecsString("registry.data.strings", value)
}

// ## related fields

// User create the ECS complain 'related.user' field.
// All the user names seen on your event.
func (nsRelated) User(value string) diag.Field {
	return ecsString("related.user", value)
}

// IP create the ECS complain 'related.ip' field.
// All of the IPs seen on your event.
func (nsRelated) IP(value string) diag.Field {
	return ecsString("related.ip", value)
}

// ## rule fields

// ID create the ECS complain 'rule.id' field.
// A rule ID that is unique within the scope of an agent, observer, or
// other entity using the rule for detection of this event.
func (nsRule) ID(value string) diag.Field {
	return ecsString("rule.id", value)
}

// Version create the ECS complain 'rule.version' field.
// The version / revision of the rule being used for analysis.
func (nsRule) Version(value string) diag.Field {
	return ecsString("rule.version", value)
}

// Name create the ECS complain 'rule.name' field.
// The name of the rule or signature generating the event.
func (nsRule) Name(value string) diag.Field {
	return ecsString("rule.name", value)
}

// Description create the ECS complain 'rule.description' field.
// The description of the rule generating the event.
func (nsRule) Description(value string) diag.Field {
	return ecsString("rule.description", value)
}

// UUID create the ECS complain 'rule.uuid' field.
// A rule ID that is unique within the scope of a set or group of agents,
// observers, or other entities using the rule for detection of this
// event.
func (nsRule) UUID(value string) diag.Field {
	return ecsString("rule.uuid", value)
}

// Category create the ECS complain 'rule.category' field.
// A categorization value keyword used by the entity using the rule for
// detection of this event.
func (nsRule) Category(value string) diag.Field {
	return ecsString("rule.category", value)
}

// Reference create the ECS complain 'rule.reference' field.
// Reference URL to additional information about the rule used to generate
// this event. The URL can point to the vendor's documentation about the
// rule. If that's not available, it can also be a link to a more general
// page describing this type of alert.
func (nsRule) Reference(value string) diag.Field {
	return ecsString("rule.reference", value)
}

// Ruleset create the ECS complain 'rule.ruleset' field.
// Name of the ruleset, policy, group, or parent category in which the
// rule used to generate this event is a member.
func (nsRule) Ruleset(value string) diag.Field {
	return ecsString("rule.ruleset", value)
}

// ## server fields

// Packets create the ECS complain 'server.packets' field.
// Packets sent from the server to the client.
func (nsServer) Packets(value int64) diag.Field {
	return ecsInt64("server.packets", value)
}

// IP create the ECS complain 'server.ip' field.
// IP address of the server. Can be one or multiple IPv4 or IPv6
// addresses.
func (nsServer) IP(value string) diag.Field {
	return ecsString("server.ip", value)
}

// RegisteredDomain create the ECS complain 'server.registered_domain' field.
// The highest registered server domain, stripped of the subdomain. For
// example, the registered domain for "foo.google.com" is "google.com".
// This value can be determined precisely with a list like the public
// suffix list (http://publicsuffix.org). Trying to approximate this by
// simply taking the last two labels will not work well for TLDs such as
// "co.uk".
func (nsServer) RegisteredDomain(value string) diag.Field {
	return ecsString("server.registered_domain", value)
}

// Address create the ECS complain 'server.address' field.
// Some event server addresses are defined ambiguously. The event will
// sometimes list an IP, a domain or a unix socket.  You should always
// store the raw address in the `.address` field. Then it should be
// duplicated to `.ip` or `.domain`, depending on which one it is.
func (nsServer) Address(value string) diag.Field {
	return ecsString("server.address", value)
}

// Domain create the ECS complain 'server.domain' field.
// Server domain.
func (nsServer) Domain(value string) diag.Field {
	return ecsString("server.domain", value)
}

// TopLevelDomain create the ECS complain 'server.top_level_domain' field.
// The effective top level domain (eTLD), also known as the domain suffix,
// is the last part of the domain name. For example, the top level domain
// for google.com is "com". This value can be determined precisely with a
// list like the public suffix list (http://publicsuffix.org). Trying to
// approximate this by simply taking the last label will not work well for
// effective TLDs such as "co.uk".
func (nsServer) TopLevelDomain(value string) diag.Field {
	return ecsString("server.top_level_domain", value)
}

// MAC create the ECS complain 'server.mac' field.
// MAC address of the server.
func (nsServer) MAC(value string) diag.Field {
	return ecsString("server.mac", value)
}

// Port create the ECS complain 'server.port' field.
// Port of the server.
func (nsServer) Port(value int64) diag.Field {
	return ecsInt64("server.port", value)
}

// Bytes create the ECS complain 'server.bytes' field.
// Bytes sent from the server to the client.
func (nsServer) Bytes(value int64) diag.Field {
	return ecsInt64("server.bytes", value)
}

// ## server.as fields

// Number create the ECS complain 'server.as.number' field.
// Unique number allocated to the autonomous system. The autonomous system
// number (ASN) uniquely identifies each network on the Internet.
func (nsServerAs) Number(value int64) diag.Field {
	return ecsInt64("server.as.number", value)
}

// ## server.as.organization fields

// Name create the ECS complain 'server.as.organization.name' field.
// Organization name.
func (nsServerAsOrganization) Name(value string) diag.Field {
	return ecsString("server.as.organization.name", value)
}

// ## server.geo fields

// CountryIsoCode create the ECS complain 'server.geo.country_iso_code' field.
// Country ISO code.
func (nsServerGeo) CountryIsoCode(value string) diag.Field {
	return ecsString("server.geo.country_iso_code", value)
}

// RegionName create the ECS complain 'server.geo.region_name' field.
// Region name.
func (nsServerGeo) RegionName(value string) diag.Field {
	return ecsString("server.geo.region_name", value)
}

// RegionIsoCode create the ECS complain 'server.geo.region_iso_code' field.
// Region ISO code.
func (nsServerGeo) RegionIsoCode(value string) diag.Field {
	return ecsString("server.geo.region_iso_code", value)
}

// CountryName create the ECS complain 'server.geo.country_name' field.
// Country name.
func (nsServerGeo) CountryName(value string) diag.Field {
	return ecsString("server.geo.country_name", value)
}

// Location create the ECS complain 'server.geo.location' field.
// Longitude and latitude.
func (nsServerGeo) Location(value string) diag.Field {
	return ecsString("server.geo.location", value)
}

// ContinentName create the ECS complain 'server.geo.continent_name' field.
// Name of the continent.
func (nsServerGeo) ContinentName(value string) diag.Field {
	return ecsString("server.geo.continent_name", value)
}

// CityName create the ECS complain 'server.geo.city_name' field.
// City name.
func (nsServerGeo) CityName(value string) diag.Field {
	return ecsString("server.geo.city_name", value)
}

// Name create the ECS complain 'server.geo.name' field.
// User-defined description of a location, at the level of granularity
// they care about. Could be the name of their data centers, the floor
// number, if this describes a local physical entity, city names. Not
// typically used in automated geolocation.
func (nsServerGeo) Name(value string) diag.Field {
	return ecsString("server.geo.name", value)
}

// ## server.nat fields

// IP create the ECS complain 'server.nat.ip' field.
// Translated ip of destination based NAT sessions (e.g. internet to
// private DMZ) Typically used with load balancers, firewalls, or routers.
func (nsServerNat) IP(value string) diag.Field {
	return ecsString("server.nat.ip", value)
}

// Port create the ECS complain 'server.nat.port' field.
// Translated port of destination based NAT sessions (e.g. internet to
// private DMZ) Typically used with load balancers, firewalls, or routers.
func (nsServerNat) Port(value int64) diag.Field {
	return ecsInt64("server.nat.port", value)
}

// ## server.user fields

// Name create the ECS complain 'server.user.name' field.
// Short name or login of the user.
func (nsServerUser) Name(value string) diag.Field {
	return ecsString("server.user.name", value)
}

// Hash create the ECS complain 'server.user.hash' field.
// Unique user hash to correlate information for a user in anonymized
// form. Useful if `user.id` or `user.name` contain confidential
// information and cannot be used.
func (nsServerUser) Hash(value string) diag.Field {
	return ecsString("server.user.hash", value)
}

// ID create the ECS complain 'server.user.id' field.
// One or multiple unique identifiers of the user.
func (nsServerUser) ID(value string) diag.Field {
	return ecsString("server.user.id", value)
}

// Domain create the ECS complain 'server.user.domain' field.
// Name of the directory the user is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsServerUser) Domain(value string) diag.Field {
	return ecsString("server.user.domain", value)
}

// Email create the ECS complain 'server.user.email' field.
// User email address.
func (nsServerUser) Email(value string) diag.Field {
	return ecsString("server.user.email", value)
}

// FullName create the ECS complain 'server.user.full_name' field.
// User's full name, if available.
func (nsServerUser) FullName(value string) diag.Field {
	return ecsString("server.user.full_name", value)
}

// ## server.user.group fields

// Name create the ECS complain 'server.user.group.name' field.
// Name of the group.
func (nsServerUserGroup) Name(value string) diag.Field {
	return ecsString("server.user.group.name", value)
}

// Domain create the ECS complain 'server.user.group.domain' field.
// Name of the directory the group is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsServerUserGroup) Domain(value string) diag.Field {
	return ecsString("server.user.group.domain", value)
}

// ID create the ECS complain 'server.user.group.id' field.
// Unique identifier for the group on the system/platform.
func (nsServerUserGroup) ID(value string) diag.Field {
	return ecsString("server.user.group.id", value)
}

// ## service fields

// Name create the ECS complain 'service.name' field.
// Name of the service data is collected from. The name of the service is
// normally user given. This allows for distributed services that run on
// multiple hosts to correlate the related instances based on the name. In
// the case of Elasticsearch the `service.name` could contain the cluster
// name. For Beats the `service.name` is by default a copy of the
// `service.type` field if no name is specified.
func (nsService) Name(value string) diag.Field {
	return ecsString("service.name", value)
}

// Version create the ECS complain 'service.version' field.
// Version of the service the data was collected from. This allows to look
// at a data set only for a specific version of a service.
func (nsService) Version(value string) diag.Field {
	return ecsString("service.version", value)
}

// Type create the ECS complain 'service.type' field.
// The type of the service data is collected from. The type can be used to
// group and correlate logs and metrics from one service type. Example: If
// logs or metrics are collected from Elasticsearch, `service.type` would
// be `elasticsearch`.
func (nsService) Type(value string) diag.Field {
	return ecsString("service.type", value)
}

// State create the ECS complain 'service.state' field.
// Current state of the service.
func (nsService) State(value string) diag.Field {
	return ecsString("service.state", value)
}

// ID create the ECS complain 'service.id' field.
// Unique identifier of the running service. If the service is comprised
// of many nodes, the `service.id` should be the same for all nodes. This
// id should uniquely identify the service. This makes it possible to
// correlate logs and metrics for one specific service, no matter which
// particular node emitted the event. Note that if you need to see the
// events from one specific host of the service, you should filter on that
// `host.name` or `host.id` instead.
func (nsService) ID(value string) diag.Field {
	return ecsString("service.id", value)
}

// EphemeralID create the ECS complain 'service.ephemeral_id' field.
// Ephemeral identifier of this service (if one exists). This id normally
// changes across restarts, but `service.id` does not.
func (nsService) EphemeralID(value string) diag.Field {
	return ecsString("service.ephemeral_id", value)
}

// ## service.node fields

// Name create the ECS complain 'service.node.name' field.
// Name of a service node. This allows for two nodes of the same service
// running on the same host to be differentiated. Therefore,
// `service.node.name` should typically be unique across nodes of a given
// service. In the case of Elasticsearch, the `service.node.name` could
// contain the unique node name within the Elasticsearch cluster. In cases
// where the service doesn't have the concept of a node name, the host
// name or container name can be used to distinguish running instances
// that make up this service. If those do not provide uniqueness (e.g.
// multiple instances of the service running on the same host) - the node
// name can be manually set.
func (nsServiceNode) Name(value string) diag.Field {
	return ecsString("service.node.name", value)
}

// ## source fields

// IP create the ECS complain 'source.ip' field.
// IP address of the source. Can be one or multiple IPv4 or IPv6
// addresses.
func (nsSource) IP(value string) diag.Field {
	return ecsString("source.ip", value)
}

// Port create the ECS complain 'source.port' field.
// Port of the source.
func (nsSource) Port(value int64) diag.Field {
	return ecsInt64("source.port", value)
}

// Domain create the ECS complain 'source.domain' field.
// Source domain.
func (nsSource) Domain(value string) diag.Field {
	return ecsString("source.domain", value)
}

// Bytes create the ECS complain 'source.bytes' field.
// Bytes sent from the source to the destination.
func (nsSource) Bytes(value int64) diag.Field {
	return ecsInt64("source.bytes", value)
}

// TopLevelDomain create the ECS complain 'source.top_level_domain' field.
// The effective top level domain (eTLD), also known as the domain suffix,
// is the last part of the domain name. For example, the top level domain
// for google.com is "com". This value can be determined precisely with a
// list like the public suffix list (http://publicsuffix.org). Trying to
// approximate this by simply taking the last label will not work well for
// effective TLDs such as "co.uk".
func (nsSource) TopLevelDomain(value string) diag.Field {
	return ecsString("source.top_level_domain", value)
}

// RegisteredDomain create the ECS complain 'source.registered_domain' field.
// The highest registered source domain, stripped of the subdomain. For
// example, the registered domain for "foo.google.com" is "google.com".
// This value can be determined precisely with a list like the public
// suffix list (http://publicsuffix.org). Trying to approximate this by
// simply taking the last two labels will not work well for TLDs such as
// "co.uk".
func (nsSource) RegisteredDomain(value string) diag.Field {
	return ecsString("source.registered_domain", value)
}

// Packets create the ECS complain 'source.packets' field.
// Packets sent from the source to the destination.
func (nsSource) Packets(value int64) diag.Field {
	return ecsInt64("source.packets", value)
}

// MAC create the ECS complain 'source.mac' field.
// MAC address of the source.
func (nsSource) MAC(value string) diag.Field {
	return ecsString("source.mac", value)
}

// Address create the ECS complain 'source.address' field.
// Some event source addresses are defined ambiguously. The event will
// sometimes list an IP, a domain or a unix socket.  You should always
// store the raw address in the `.address` field. Then it should be
// duplicated to `.ip` or `.domain`, depending on which one it is.
func (nsSource) Address(value string) diag.Field {
	return ecsString("source.address", value)
}

// ## source.as fields

// Number create the ECS complain 'source.as.number' field.
// Unique number allocated to the autonomous system. The autonomous system
// number (ASN) uniquely identifies each network on the Internet.
func (nsSourceAs) Number(value int64) diag.Field {
	return ecsInt64("source.as.number", value)
}

// ## source.as.organization fields

// Name create the ECS complain 'source.as.organization.name' field.
// Organization name.
func (nsSourceAsOrganization) Name(value string) diag.Field {
	return ecsString("source.as.organization.name", value)
}

// ## source.geo fields

// RegionIsoCode create the ECS complain 'source.geo.region_iso_code' field.
// Region ISO code.
func (nsSourceGeo) RegionIsoCode(value string) diag.Field {
	return ecsString("source.geo.region_iso_code", value)
}

// RegionName create the ECS complain 'source.geo.region_name' field.
// Region name.
func (nsSourceGeo) RegionName(value string) diag.Field {
	return ecsString("source.geo.region_name", value)
}

// Name create the ECS complain 'source.geo.name' field.
// User-defined description of a location, at the level of granularity
// they care about. Could be the name of their data centers, the floor
// number, if this describes a local physical entity, city names. Not
// typically used in automated geolocation.
func (nsSourceGeo) Name(value string) diag.Field {
	return ecsString("source.geo.name", value)
}

// ContinentName create the ECS complain 'source.geo.continent_name' field.
// Name of the continent.
func (nsSourceGeo) ContinentName(value string) diag.Field {
	return ecsString("source.geo.continent_name", value)
}

// CityName create the ECS complain 'source.geo.city_name' field.
// City name.
func (nsSourceGeo) CityName(value string) diag.Field {
	return ecsString("source.geo.city_name", value)
}

// Location create the ECS complain 'source.geo.location' field.
// Longitude and latitude.
func (nsSourceGeo) Location(value string) diag.Field {
	return ecsString("source.geo.location", value)
}

// CountryName create the ECS complain 'source.geo.country_name' field.
// Country name.
func (nsSourceGeo) CountryName(value string) diag.Field {
	return ecsString("source.geo.country_name", value)
}

// CountryIsoCode create the ECS complain 'source.geo.country_iso_code' field.
// Country ISO code.
func (nsSourceGeo) CountryIsoCode(value string) diag.Field {
	return ecsString("source.geo.country_iso_code", value)
}

// ## source.nat fields

// IP create the ECS complain 'source.nat.ip' field.
// Translated ip of source based NAT sessions (e.g. internal client to
// internet) Typically connections traversing load balancers, firewalls,
// or routers.
func (nsSourceNat) IP(value string) diag.Field {
	return ecsString("source.nat.ip", value)
}

// Port create the ECS complain 'source.nat.port' field.
// Translated port of source based NAT sessions. (e.g. internal client to
// internet) Typically used with load balancers, firewalls, or routers.
func (nsSourceNat) Port(value int64) diag.Field {
	return ecsInt64("source.nat.port", value)
}

// ## source.user fields

// Email create the ECS complain 'source.user.email' field.
// User email address.
func (nsSourceUser) Email(value string) diag.Field {
	return ecsString("source.user.email", value)
}

// Name create the ECS complain 'source.user.name' field.
// Short name or login of the user.
func (nsSourceUser) Name(value string) diag.Field {
	return ecsString("source.user.name", value)
}

// FullName create the ECS complain 'source.user.full_name' field.
// User's full name, if available.
func (nsSourceUser) FullName(value string) diag.Field {
	return ecsString("source.user.full_name", value)
}

// ID create the ECS complain 'source.user.id' field.
// One or multiple unique identifiers of the user.
func (nsSourceUser) ID(value string) diag.Field {
	return ecsString("source.user.id", value)
}

// Domain create the ECS complain 'source.user.domain' field.
// Name of the directory the user is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsSourceUser) Domain(value string) diag.Field {
	return ecsString("source.user.domain", value)
}

// Hash create the ECS complain 'source.user.hash' field.
// Unique user hash to correlate information for a user in anonymized
// form. Useful if `user.id` or `user.name` contain confidential
// information and cannot be used.
func (nsSourceUser) Hash(value string) diag.Field {
	return ecsString("source.user.hash", value)
}

// ## source.user.group fields

// ID create the ECS complain 'source.user.group.id' field.
// Unique identifier for the group on the system/platform.
func (nsSourceUserGroup) ID(value string) diag.Field {
	return ecsString("source.user.group.id", value)
}

// Domain create the ECS complain 'source.user.group.domain' field.
// Name of the directory the group is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsSourceUserGroup) Domain(value string) diag.Field {
	return ecsString("source.user.group.domain", value)
}

// Name create the ECS complain 'source.user.group.name' field.
// Name of the group.
func (nsSourceUserGroup) Name(value string) diag.Field {
	return ecsString("source.user.group.name", value)
}

// ## threat fields

// Framework create the ECS complain 'threat.framework' field.
// Name of the threat framework used to further categorize and classify
// the tactic and technique of the reported threat. Framework
// classification can be provided by detecting systems, evaluated at
// ingest time, or retrospectively tagged to events.
func (nsThreat) Framework(value string) diag.Field {
	return ecsString("threat.framework", value)
}

// ## threat.tactic fields

// ID create the ECS complain 'threat.tactic.id' field.
// The id of tactic used by this threat. You can use the Mitre ATT&CK
// Matrix Tactic categorization, for example. (ex.
// https://attack.mitre.org/tactics/TA0040/ )
func (nsThreatTactic) ID(value string) diag.Field {
	return ecsString("threat.tactic.id", value)
}

// Reference create the ECS complain 'threat.tactic.reference' field.
// The reference url of tactic used by this threat. You can use the Mitre
// ATT&CK Matrix Tactic categorization, for example. (ex.
// https://attack.mitre.org/tactics/TA0040/ )
func (nsThreatTactic) Reference(value string) diag.Field {
	return ecsString("threat.tactic.reference", value)
}

// Name create the ECS complain 'threat.tactic.name' field.
// Name of the type of tactic used by this threat. You can use the Mitre
// ATT&CK Matrix Tactic categorization, for example. (ex.
// https://attack.mitre.org/tactics/TA0040/ )
func (nsThreatTactic) Name(value string) diag.Field {
	return ecsString("threat.tactic.name", value)
}

// ## threat.technique fields

// Reference create the ECS complain 'threat.technique.reference' field.
// The reference url of technique used by this tactic. You can use the
// Mitre ATT&CK Matrix Tactic categorization, for example. (ex.
// https://attack.mitre.org/techniques/T1499/ )
func (nsThreatTechnique) Reference(value string) diag.Field {
	return ecsString("threat.technique.reference", value)
}

// Name create the ECS complain 'threat.technique.name' field.
// The name of technique used by this tactic. You can use the Mitre ATT&CK
// Matrix Tactic categorization, for example. (ex.
// https://attack.mitre.org/techniques/T1499/ )
func (nsThreatTechnique) Name(value string) diag.Field {
	return ecsString("threat.technique.name", value)
}

// ID create the ECS complain 'threat.technique.id' field.
// The id of technique used by this tactic. You can use the Mitre ATT&CK
// Matrix Tactic categorization, for example. (ex.
// https://attack.mitre.org/techniques/T1499/ )
func (nsThreatTechnique) ID(value string) diag.Field {
	return ecsString("threat.technique.id", value)
}

// ## tls fields

// VersionProtocol create the ECS complain 'tls.version_protocol' field.
// Normalized lowercase protocol name parsed from original string.
func (nsTLS) VersionProtocol(value string) diag.Field {
	return ecsString("tls.version_protocol", value)
}

// Cipher create the ECS complain 'tls.cipher' field.
// String indicating the cipher used during the current connection.
func (nsTLS) Cipher(value string) diag.Field {
	return ecsString("tls.cipher", value)
}

// Established create the ECS complain 'tls.established' field.
// Boolean flag indicating if the TLS negotiation was successful and
// transitioned to an encrypted tunnel.
func (nsTLS) Established(value bool) diag.Field {
	return ecsBool("tls.established", value)
}

// Curve create the ECS complain 'tls.curve' field.
// String indicating the curve used for the given cipher, when applicable.
func (nsTLS) Curve(value string) diag.Field {
	return ecsString("tls.curve", value)
}

// NextProtocol create the ECS complain 'tls.next_protocol' field.
// String indicating the protocol being tunneled. Per the values in the
// IANA registry
// (https://www.iana.org/assignments/tls-extensiontype-values/tls-extensiontype-values.xhtml#alpn-protocol-ids),
// this string should be lower case.
func (nsTLS) NextProtocol(value string) diag.Field {
	return ecsString("tls.next_protocol", value)
}

// Version create the ECS complain 'tls.version' field.
// Numeric part of the version parsed from the original string.
func (nsTLS) Version(value string) diag.Field {
	return ecsString("tls.version", value)
}

// Resumed create the ECS complain 'tls.resumed' field.
// Boolean flag indicating if this TLS connection was resumed from an
// existing TLS negotiation.
func (nsTLS) Resumed(value bool) diag.Field {
	return ecsBool("tls.resumed", value)
}

// ## tls.client fields

// CertificateChain create the ECS complain 'tls.client.certificate_chain' field.
// Array of PEM-encoded certificates that make up the certificate chain
// offered by the client. This is usually mutually-exclusive of
// `client.certificate` since that value should be the first certificate
// in the chain.
func (nsTLSClient) CertificateChain(value string) diag.Field {
	return ecsString("tls.client.certificate_chain", value)
}

// SupportedCiphers create the ECS complain 'tls.client.supported_ciphers' field.
// Array of ciphers offered by the client during the client hello.
func (nsTLSClient) SupportedCiphers(value string) diag.Field {
	return ecsString("tls.client.supported_ciphers", value)
}

// NotAfter create the ECS complain 'tls.client.not_after' field.
// Date/Time indicating when client certificate is no longer considered
// valid.
func (nsTLSClient) NotAfter(value time.Time) diag.Field {
	return ecsTime("tls.client.not_after", value)
}

// Subject create the ECS complain 'tls.client.subject' field.
// Distinguished name of subject of the x.509 certificate presented by the
// client.
func (nsTLSClient) Subject(value string) diag.Field {
	return ecsString("tls.client.subject", value)
}

// Ja3 create the ECS complain 'tls.client.ja3' field.
// A hash that identifies clients based on how they perform an SSL/TLS
// handshake.
func (nsTLSClient) Ja3(value string) diag.Field {
	return ecsString("tls.client.ja3", value)
}

// NotBefore create the ECS complain 'tls.client.not_before' field.
// Date/Time indicating when client certificate is first considered valid.
func (nsTLSClient) NotBefore(value time.Time) diag.Field {
	return ecsTime("tls.client.not_before", value)
}

// Certificate create the ECS complain 'tls.client.certificate' field.
// PEM-encoded stand-alone certificate offered by the client. This is
// usually mutually-exclusive of `client.certificate_chain` since this
// value also exists in that list.
func (nsTLSClient) Certificate(value string) diag.Field {
	return ecsString("tls.client.certificate", value)
}

// ServerName create the ECS complain 'tls.client.server_name' field.
// Also called an SNI, this tells the server which hostname to which the
// client is attempting to connect. When this value is available, it
// should get copied to `destination.domain`.
func (nsTLSClient) ServerName(value string) diag.Field {
	return ecsString("tls.client.server_name", value)
}

// Issuer create the ECS complain 'tls.client.issuer' field.
// Distinguished name of subject of the issuer of the x.509 certificate
// presented by the client.
func (nsTLSClient) Issuer(value string) diag.Field {
	return ecsString("tls.client.issuer", value)
}

// ## tls.client.hash fields

// Sha256 create the ECS complain 'tls.client.hash.sha256' field.
// Certificate fingerprint using the SHA256 digest of DER-encoded version
// of certificate offered by the client. For consistency with other hash
// values, this value should be formatted as an uppercase hash.
func (nsTLSClientHash) Sha256(value string) diag.Field {
	return ecsString("tls.client.hash.sha256", value)
}

// Md5 create the ECS complain 'tls.client.hash.md5' field.
// Certificate fingerprint using the MD5 digest of DER-encoded version of
// certificate offered by the client. For consistency with other hash
// values, this value should be formatted as an uppercase hash.
func (nsTLSClientHash) Md5(value string) diag.Field {
	return ecsString("tls.client.hash.md5", value)
}

// Sha1 create the ECS complain 'tls.client.hash.sha1' field.
// Certificate fingerprint using the SHA1 digest of DER-encoded version of
// certificate offered by the client. For consistency with other hash
// values, this value should be formatted as an uppercase hash.
func (nsTLSClientHash) Sha1(value string) diag.Field {
	return ecsString("tls.client.hash.sha1", value)
}

// ## tls.server fields

// NotBefore create the ECS complain 'tls.server.not_before' field.
// Timestamp indicating when server certificate is first considered valid.
func (nsTLSServer) NotBefore(value time.Time) diag.Field {
	return ecsTime("tls.server.not_before", value)
}

// CertificateChain create the ECS complain 'tls.server.certificate_chain' field.
// Array of PEM-encoded certificates that make up the certificate chain
// offered by the server. This is usually mutually-exclusive of
// `server.certificate` since that value should be the first certificate
// in the chain.
func (nsTLSServer) CertificateChain(value string) diag.Field {
	return ecsString("tls.server.certificate_chain", value)
}

// NotAfter create the ECS complain 'tls.server.not_after' field.
// Timestamp indicating when server certificate is no longer considered
// valid.
func (nsTLSServer) NotAfter(value time.Time) diag.Field {
	return ecsTime("tls.server.not_after", value)
}

// Subject create the ECS complain 'tls.server.subject' field.
// Subject of the x.509 certificate presented by the server.
func (nsTLSServer) Subject(value string) diag.Field {
	return ecsString("tls.server.subject", value)
}

// Certificate create the ECS complain 'tls.server.certificate' field.
// PEM-encoded stand-alone certificate offered by the server. This is
// usually mutually-exclusive of `server.certificate_chain` since this
// value also exists in that list.
func (nsTLSServer) Certificate(value string) diag.Field {
	return ecsString("tls.server.certificate", value)
}

// Issuer create the ECS complain 'tls.server.issuer' field.
// Subject of the issuer of the x.509 certificate presented by the server.
func (nsTLSServer) Issuer(value string) diag.Field {
	return ecsString("tls.server.issuer", value)
}

// Ja3s create the ECS complain 'tls.server.ja3s' field.
// A hash that identifies servers based on how they perform an SSL/TLS
// handshake.
func (nsTLSServer) Ja3s(value string) diag.Field {
	return ecsString("tls.server.ja3s", value)
}

// ## tls.server.hash fields

// Md5 create the ECS complain 'tls.server.hash.md5' field.
// Certificate fingerprint using the MD5 digest of DER-encoded version of
// certificate offered by the server. For consistency with other hash
// values, this value should be formatted as an uppercase hash.
func (nsTLSServerHash) Md5(value string) diag.Field {
	return ecsString("tls.server.hash.md5", value)
}

// Sha256 create the ECS complain 'tls.server.hash.sha256' field.
// Certificate fingerprint using the SHA256 digest of DER-encoded version
// of certificate offered by the server. For consistency with other hash
// values, this value should be formatted as an uppercase hash.
func (nsTLSServerHash) Sha256(value string) diag.Field {
	return ecsString("tls.server.hash.sha256", value)
}

// Sha1 create the ECS complain 'tls.server.hash.sha1' field.
// Certificate fingerprint using the SHA1 digest of DER-encoded version of
// certificate offered by the server. For consistency with other hash
// values, this value should be formatted as an uppercase hash.
func (nsTLSServerHash) Sha1(value string) diag.Field {
	return ecsString("tls.server.hash.sha1", value)
}

// ## trace fields

// ID create the ECS complain 'trace.id' field.
// Unique identifier of the trace. A trace groups multiple events like
// transactions that belong together. For example, a user request handled
// by multiple inter-connected services.
func (nsTrace) ID(value string) diag.Field {
	return ecsString("trace.id", value)
}

// ## transaction fields

// ID create the ECS complain 'transaction.id' field.
// Unique identifier of the transaction. A transaction is the highest
// level of work measured within a service, such as a request to a server.
func (nsTransaction) ID(value string) diag.Field {
	return ecsString("transaction.id", value)
}

// ## url fields

// Query create the ECS complain 'url.query' field.
// The query field describes the query string of the request, such as
// "q=elasticsearch". The `?` is excluded from the query string. If a URL
// contains no `?`, there is no query field. If there is a `?` but no
// query, the query field exists with an empty string. The `exists` query
// can be used to differentiate between the two cases.
func (nsURL) Query(value string) diag.Field {
	return ecsString("url.query", value)
}

// Fragment create the ECS complain 'url.fragment' field.
// Portion of the url after the `#`, such as "top". The `#` is not part of
// the fragment.
func (nsURL) Fragment(value string) diag.Field {
	return ecsString("url.fragment", value)
}

// Port create the ECS complain 'url.port' field.
// Port of the request, such as 443.
func (nsURL) Port(value int64) diag.Field {
	return ecsInt64("url.port", value)
}

// Password create the ECS complain 'url.password' field.
// Password of the request.
func (nsURL) Password(value string) diag.Field {
	return ecsString("url.password", value)
}

// RegisteredDomain create the ECS complain 'url.registered_domain' field.
// The highest registered url domain, stripped of the subdomain. For
// example, the registered domain for "foo.google.com" is "google.com".
// This value can be determined precisely with a list like the public
// suffix list (http://publicsuffix.org). Trying to approximate this by
// simply taking the last two labels will not work well for TLDs such as
// "co.uk".
func (nsURL) RegisteredDomain(value string) diag.Field {
	return ecsString("url.registered_domain", value)
}

// Scheme create the ECS complain 'url.scheme' field.
// Scheme of the request, such as "https". Note: The `:` is not part of
// the scheme.
func (nsURL) Scheme(value string) diag.Field {
	return ecsString("url.scheme", value)
}

// TopLevelDomain create the ECS complain 'url.top_level_domain' field.
// The effective top level domain (eTLD), also known as the domain suffix,
// is the last part of the domain name. For example, the top level domain
// for google.com is "com". This value can be determined precisely with a
// list like the public suffix list (http://publicsuffix.org). Trying to
// approximate this by simply taking the last label will not work well for
// effective TLDs such as "co.uk".
func (nsURL) TopLevelDomain(value string) diag.Field {
	return ecsString("url.top_level_domain", value)
}

// Full create the ECS complain 'url.full' field.
// If full URLs are important to your use case, they should be stored in
// `url.full`, whether this field is reconstructed or present in the event
// source.
func (nsURL) Full(value string) diag.Field {
	return ecsString("url.full", value)
}

// Username create the ECS complain 'url.username' field.
// Username of the request.
func (nsURL) Username(value string) diag.Field {
	return ecsString("url.username", value)
}

// Extension create the ECS complain 'url.extension' field.
// The field contains the file extension from the original request url.
// The file extension is only set if it exists, as not every url has a
// file extension. The leading period must not be included. For example,
// the value must be "png", not ".png".
func (nsURL) Extension(value string) diag.Field {
	return ecsString("url.extension", value)
}

// Original create the ECS complain 'url.original' field.
// Unmodified original url as seen in the event source. Note that in
// network monitoring, the observed URL may be a full URL, whereas in
// access logs, the URL is often just represented as a path. This field is
// meant to represent the URL as it was observed, complete or not.
func (nsURL) Original(value string) diag.Field {
	return ecsString("url.original", value)
}

// Path create the ECS complain 'url.path' field.
// Path of the request, such as "/search".
func (nsURL) Path(value string) diag.Field {
	return ecsString("url.path", value)
}

// Domain create the ECS complain 'url.domain' field.
// Domain of the url, such as "www.elastic.co". In some cases a URL may
// refer to an IP and/or port directly, without a domain name. In this
// case, the IP address would go to the `domain` field.
func (nsURL) Domain(value string) diag.Field {
	return ecsString("url.domain", value)
}

// ## user fields

// ID create the ECS complain 'user.id' field.
// One or multiple unique identifiers of the user.
func (nsUser) ID(value string) diag.Field {
	return ecsString("user.id", value)
}

// FullName create the ECS complain 'user.full_name' field.
// User's full name, if available.
func (nsUser) FullName(value string) diag.Field {
	return ecsString("user.full_name", value)
}

// Domain create the ECS complain 'user.domain' field.
// Name of the directory the user is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsUser) Domain(value string) diag.Field {
	return ecsString("user.domain", value)
}

// Email create the ECS complain 'user.email' field.
// User email address.
func (nsUser) Email(value string) diag.Field {
	return ecsString("user.email", value)
}

// Hash create the ECS complain 'user.hash' field.
// Unique user hash to correlate information for a user in anonymized
// form. Useful if `user.id` or `user.name` contain confidential
// information and cannot be used.
func (nsUser) Hash(value string) diag.Field {
	return ecsString("user.hash", value)
}

// Name create the ECS complain 'user.name' field.
// Short name or login of the user.
func (nsUser) Name(value string) diag.Field {
	return ecsString("user.name", value)
}

// ## user.group fields

// Name create the ECS complain 'user.group.name' field.
// Name of the group.
func (nsUserGroup) Name(value string) diag.Field {
	return ecsString("user.group.name", value)
}

// Domain create the ECS complain 'user.group.domain' field.
// Name of the directory the group is a member of. For example, an LDAP or
// Active Directory domain name.
func (nsUserGroup) Domain(value string) diag.Field {
	return ecsString("user.group.domain", value)
}

// ID create the ECS complain 'user.group.id' field.
// Unique identifier for the group on the system/platform.
func (nsUserGroup) ID(value string) diag.Field {
	return ecsString("user.group.id", value)
}

// ## user_agent fields

// Original create the ECS complain 'user_agent.original' field.
// Unparsed user_agent string.
func (nsUserAgent) Original(value string) diag.Field {
	return ecsString("user_agent.original", value)
}

// Name create the ECS complain 'user_agent.name' field.
// Name of the user agent.
func (nsUserAgent) Name(value string) diag.Field {
	return ecsString("user_agent.name", value)
}

// Version create the ECS complain 'user_agent.version' field.
// Version of the user agent.
func (nsUserAgent) Version(value string) diag.Field {
	return ecsString("user_agent.version", value)
}

// ## user_agent.device fields

// Name create the ECS complain 'user_agent.device.name' field.
// Name of the device.
func (nsUserAgentDevice) Name(value string) diag.Field {
	return ecsString("user_agent.device.name", value)
}

// ## user_agent.os fields

// Full create the ECS complain 'user_agent.os.full' field.
// Operating system name, including the version or code name.
func (nsUserAgentOS) Full(value string) diag.Field {
	return ecsString("user_agent.os.full", value)
}

// Version create the ECS complain 'user_agent.os.version' field.
// Operating system version as a raw string.
func (nsUserAgentOS) Version(value string) diag.Field {
	return ecsString("user_agent.os.version", value)
}

// Kernel create the ECS complain 'user_agent.os.kernel' field.
// Operating system kernel version as a raw string.
func (nsUserAgentOS) Kernel(value string) diag.Field {
	return ecsString("user_agent.os.kernel", value)
}

// Family create the ECS complain 'user_agent.os.family' field.
// OS family (such as redhat, debian, freebsd, windows).
func (nsUserAgentOS) Family(value string) diag.Field {
	return ecsString("user_agent.os.family", value)
}

// Name create the ECS complain 'user_agent.os.name' field.
// Operating system name, without the version.
func (nsUserAgentOS) Name(value string) diag.Field {
	return ecsString("user_agent.os.name", value)
}

// Platform create the ECS complain 'user_agent.os.platform' field.
// Operating system platform (such centos, ubuntu, windows).
func (nsUserAgentOS) Platform(value string) diag.Field {
	return ecsString("user_agent.os.platform", value)
}

// ## vulnerability fields

// Classification create the ECS complain 'vulnerability.classification' field.
// The classification of the vulnerability scoring system. For example
// (https://www.first.org/cvss/)
func (nsVulnerability) Classification(value string) diag.Field {
	return ecsString("vulnerability.classification", value)
}

// Description create the ECS complain 'vulnerability.description' field.
// The description of the vulnerability that provides additional context
// of the vulnerability. For example
// (https://cve.mitre.org/about/faqs.html#cve_entry_descriptions_created[Common
// Vulnerabilities and Exposure CVE description])
func (nsVulnerability) Description(value string) diag.Field {
	return ecsString("vulnerability.description", value)
}

// ReportID create the ECS complain 'vulnerability.report_id' field.
// The report or scan identification number.
func (nsVulnerability) ReportID(value string) diag.Field {
	return ecsString("vulnerability.report_id", value)
}

// ID create the ECS complain 'vulnerability.id' field.
// The identification (ID) is the number portion of a vulnerability entry.
// It includes a unique identification number for the vulnerability. For
// example (https://cve.mitre.org/about/faqs.html#what_is_cve_id)[Common
// Vulnerabilities and Exposure CVE ID]
func (nsVulnerability) ID(value string) diag.Field {
	return ecsString("vulnerability.id", value)
}

// Severity create the ECS complain 'vulnerability.severity' field.
// The severity of the vulnerability can help with metrics and internal
// prioritization regarding remediation. For example
// (https://nvd.nist.gov/vuln-metrics/cvss)
func (nsVulnerability) Severity(value string) diag.Field {
	return ecsString("vulnerability.severity", value)
}

// Enumeration create the ECS complain 'vulnerability.enumeration' field.
// The type of identifier used for this vulnerability. For example
// (https://cve.mitre.org/about/)
func (nsVulnerability) Enumeration(value string) diag.Field {
	return ecsString("vulnerability.enumeration", value)
}

// Category create the ECS complain 'vulnerability.category' field.
// The type of system or architecture that the vulnerability affects.
// These may be platform-specific (for example, Debian or SUSE) or general
// (for example, Database or Firewall). For example
// (https://qualysguard.qualys.com/qwebhelp/fo_portal/knowledgebase/vulnerability_categories.htm[Qualys
// vulnerability categories]) This field must be an array.
func (nsVulnerability) Category(value string) diag.Field {
	return ecsString("vulnerability.category", value)
}

// Reference create the ECS complain 'vulnerability.reference' field.
// A resource that provides additional information, context, and
// mitigations for the identified vulnerability.
func (nsVulnerability) Reference(value string) diag.Field {
	return ecsString("vulnerability.reference", value)
}

// ## vulnerability.scanner fields

// Vendor create the ECS complain 'vulnerability.scanner.vendor' field.
// The name of the vulnerability scanner vendor.
func (nsVulnerabilityScanner) Vendor(value string) diag.Field {
	return ecsString("vulnerability.scanner.vendor", value)
}

// ## vulnerability.score fields

// Base create the ECS complain 'vulnerability.score.base' field.
// Scores can range from 0.0 to 10.0, with 10.0 being the most severe.
// Base scores cover an assessment for exploitability metrics (attack
// vector, complexity, privileges, and user interaction), impact metrics
// (confidentiality, integrity, and availability), and scope. For example
// (https://www.first.org/cvss/specification-document)
func (nsVulnerabilityScore) Base(value float64) diag.Field {
	return ecsFloat64("vulnerability.score.base", value)
}

// Temporal create the ECS complain 'vulnerability.score.temporal' field.
// Scores can range from 0.0 to 10.0, with 10.0 being the most severe.
// Temporal scores cover an assessment for code maturity, remediation
// level, and confidence. For example
// (https://www.first.org/cvss/specification-document)
func (nsVulnerabilityScore) Temporal(value float64) diag.Field {
	return ecsFloat64("vulnerability.score.temporal", value)
}

// Environmental create the ECS complain 'vulnerability.score.environmental' field.
// Scores can range from 0.0 to 10.0, with 10.0 being the most severe.
// Environmental scores cover an assessment for any modified Base metrics,
// confidentiality, integrity, and availability requirements. For example
// (https://www.first.org/cvss/specification-document)
func (nsVulnerabilityScore) Environmental(value float64) diag.Field {
	return ecsFloat64("vulnerability.score.environmental", value)
}

// Version create the ECS complain 'vulnerability.score.version' field.
// The National Vulnerability Database (NVD) provides qualitative severity
// rankings of "Low", "Medium", and "High" for CVSS v2.0 base score ranges
// in addition to the severity ratings for CVSS v3.0 as they are defined
// in the CVSS v3.0 specification. CVSS is owned and managed by FIRST.Org,
// Inc. (FIRST), a US-based non-profit organization, whose mission is to
// help computer security incident response teams across the world. For
// example (https://nvd.nist.gov/vuln-metrics/cvss)
func (nsVulnerabilityScore) Version(value string) diag.Field {
	return ecsString("vulnerability.score.version", value)
}
