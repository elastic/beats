package meraki

// device unique identifier
type Serial string

// Device contains static device attributes (i.e. dimensions)
type Device struct {
	Address     string
	Details     map[string]string
	Firmware    string
	Imei        *float64
	LanIP       string
	Location    []*float64
	Mac         string
	Model       string
	Name        string
	NetworkID   string
	Notes       string
	ProductType string // one of ["appliance", "camera", "cellularGateway", "secureConnect", "sensor", "switch", "systemsManager", "wireless", "wirelessController"]
	Serial      string
	Tags        []string
}
