package globalprotect_stats

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
	Gateways           []Gateway `xml:"Gateway"`
	TotalCurrentUsers  int       `xml:"TotalCurrentUsers"`
	TotalPreviousUsers int       `xml:"TotalPreviousUsers"`
}

type Gateway struct {
	Name          string `xml:"name"`
	CurrentUsers  int    `xml:"CurrentUsers"`
	PreviousUsers int    `xml:"PreviousUsers"`
}
