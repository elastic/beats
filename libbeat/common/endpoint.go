package common

// Endpoint represents an endpoint in the communication.
type Endpoint struct {
	IP      string
	Port    uint16
	Name    string
	Cmdline string
	Proc    string
}

// MakeEndpointPair returns source and destination endpoints from a TCP or IP tuple
// and a command-line tuple.
func MakeEndpointPair(tuple BaseTuple, cmdlineTuple *CmdlineTuple) (src Endpoint, dst Endpoint) {
	src = Endpoint{
		IP:      tuple.SrcIP.String(),
		Port:    tuple.SrcPort,
		Proc:    string(cmdlineTuple.Src),
		Cmdline: string(cmdlineTuple.SrcCommand),
	}
	dst = Endpoint{
		IP:      tuple.DstIP.String(),
		Port:    tuple.DstPort,
		Proc:    string(cmdlineTuple.Dst),
		Cmdline: string(cmdlineTuple.DstCommand),
	}
	return src, dst
}
