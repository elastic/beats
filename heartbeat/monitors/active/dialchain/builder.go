package dialchain

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

// Builder maintains a DialerChain for building dialers and dialer based
// monitoring jobs.
// The builder ensures a constant address is being used, for any host
// configured. This ensures the upper network layers (e.g. TLS) correctly see
// and process the original hostname.
type Builder struct {
	template         *DialerChain
	addrIndex        int
	resolveViaSocks5 bool
}

// BuilderSettings configures the layers of the dialer chain to be constructed
// by a Builder.
type BuilderSettings struct {
	Timeout time.Duration
	Socks5  transport.ProxyConfig
	TLS     *transport.TLSConfig
}

// Endpoint configures a host with all port numbers to be monitored by a dialer
// based job.
type Endpoint struct {
	Host  string
	Ports []uint16
}

// NewBuilder creates a new Builder for constructing dialers.
func NewBuilder(settings BuilderSettings) (*Builder, error) {
	d := &DialerChain{
		Net: netDialer(settings.Timeout),
	}
	resolveViaSocks5 := false
	withProxy := settings.Socks5.URL != ""
	if withProxy {
		d.AddLayer(SOCKS5Layer(&settings.Socks5))
		resolveViaSocks5 = !settings.Socks5.LocalResolve
	}

	// insert empty placeholder, so address can be replaced in dialer chain
	// by replacing this placeholder dialer
	idx := len(d.Layers)
	d.AddLayer(IDLayer())

	// add tls layer doing the TLS handshake based on the original address
	if tls := settings.TLS; tls != nil {
		d.AddLayer(TLSLayer(tls, settings.Timeout))
	}

	// validate dialerchain
	if err := d.TestBuild(); err != nil {
		return nil, err
	}

	return &Builder{
		template:         d,
		addrIndex:        idx,
		resolveViaSocks5: resolveViaSocks5,
	}, nil
}

// AddLayer adds another custom network layer to the dialer chain.
func (b *Builder) AddLayer(l Layer) {
	b.template.AddLayer(l)
}

// Build create a new dialer, that will always use the constant address, no matter
// which address is used to connect using the dialer.
// The dialer chain will add per layer information to the given event.
func (b *Builder) Build(addr string, event common.MapStr) (transport.Dialer, error) {
	// clone template, as multiple instance of a dialer can exist at the same time
	dchain := b.template.Clone()

	// fix the final dialers TCP-level address
	dchain.Layers[b.addrIndex] = ConstAddrLayer(addr)

	// create dialer chain with event to add per network layer information
	d, err := dchain.Build(event)
	return d, err
}

// Run executes the given function with a new dialer instance.
func (b *Builder) Run(
	addr string,
	fn func(transport.Dialer) (common.MapStr, error),
) (common.MapStr, error) {
	event := common.MapStr{}
	dialer, err := b.Build(addr, event)
	if err != nil {
		return nil, err
	}

	results, err := fn(dialer)
	event.DeepUpdate(results)
	return event, err
}

// MakeDialerJobs creates a set of monitoring jobs. The jobs behavior depends
// on the builder, endpoint and mode configurations, normally set by user
// configuration.  The task to execute the actual 'ping' receives the dialer
// and the address pair (<hostname>:<port>), required to be used, to ping the
// correctly resolved endpoint.
func MakeDialerJobs(
	b *Builder,
	typ, scheme string,
	endpoints []Endpoint,
	mode monitors.IPSettings,
	fn func(dialer transport.Dialer, addr string) (common.MapStr, error),
) ([]monitors.Job, error) {
	var jobs []monitors.Job
	for _, endpoint := range endpoints {
		endpointJobs, err := makeEndpointJobs(b, typ, scheme, endpoint, mode, fn)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, endpointJobs...)
	}

	return jobs, nil
}

func makeEndpointJobs(
	b *Builder,
	typ, scheme string,
	endpoint Endpoint,
	mode monitors.IPSettings,
	fn func(transport.Dialer, string) (common.MapStr, error),
) ([]monitors.Job, error) {

	fields := common.MapStr{
		"monitor": common.MapStr{
			"host":   endpoint.Host,
			"scheme": scheme,
		},
	}

	// Check if SOCKS5 is configured, with relying on the socks5 proxy
	// in resolving the actual IP.
	// Create one job for every port number configured.
	if b.resolveViaSocks5 {
		jobs := make([]monitors.Job, len(endpoint.Ports))
		for i, port := range endpoint.Ports {
			jobName := jobName(typ, scheme, endpoint.Host, []uint16{port})
			address := net.JoinHostPort(endpoint.Host, strconv.Itoa(int(port)))
			settings := monitors.MakeJobSetting(jobName).WithFields(fields)
			jobs[i] = monitors.MakeSimpleJob(settings, func() (common.MapStr, error) {
				return b.Run(address, func(dialer transport.Dialer) (common.MapStr, error) {
					return fn(dialer, address)
				})
			})
		}
		return jobs, nil
	}

	// Create job that first resolves one or multiple IP (depending on
	// config.Mode) in order to create one continuation Task per IP.
	jobName := jobName(typ, scheme, endpoint.Host, endpoint.Ports)
	settings := monitors.MakeHostJobSettings(jobName, endpoint.Host, mode).WithFields(fields)
	job, err := monitors.MakeByHostJob(settings,
		monitors.MakePingAllIPPortFactory(endpoint.Ports,
			func(ip *net.IPAddr, port uint16) (common.MapStr, error) {
				// use address from resolved IP
				portStr := strconv.Itoa(int(port))
				ipAddr := net.JoinHostPort(ip.String(), portStr)
				hostAddr := net.JoinHostPort(endpoint.Host, portStr)
				return b.Run(ipAddr, func(dialer transport.Dialer) (common.MapStr, error) {
					return fn(dialer, hostAddr)
				})
			}))
	if err != nil {
		return nil, err
	}
	return []monitors.Job{job}, nil
}

func jobName(typ, jobType, host string, ports []uint16) string {
	var h string
	if len(ports) == 1 {
		h = fmt.Sprintf("%v:%v", host, ports[0])
	} else {
		h = fmt.Sprintf("%v:%v", host, ports)
	}
	return fmt.Sprintf("%v-%v@%v", typ, jobType, h)
}
