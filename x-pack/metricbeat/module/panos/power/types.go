package power

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
	Power Power `xml:"power"`
}

type Power struct {
	Slots []Slot `xml:",any"`
}

type Slot struct {
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	Slot         int     `xml:"slot"`
	Description  string  `xml:"description"`
	Alarm        bool    `xml:"alarm"`
	Volts        float64 `xml:"Volts"`
	MinimumVolts float64 `xml:"min"`
	MaximumVolts float64 `xml:"max"`
}
