package logical

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
	Ifnet Ifnet `xml:"ifnet"`
}

type Ifnet struct {
	LogicalInterfaces []LogicalInterface `xml:"entry"`
}

type LogicalInterface struct {
	Name    string `xml:"name"`
	ID      int    `xml:"id"`
	Tag     int    `xml:"tag"`
	Vsys    int    `xml:"vsys"`
	Zone    string `xml:"zone"`
	Fwd     string `xml:"fwd"`
	IP      string `xml:"ip"`
	Addr    string `xml:"addr"`
	DynAddr string `xml:"dyn-addr"`
	Addr6   string `xml:"addr6"`
}
