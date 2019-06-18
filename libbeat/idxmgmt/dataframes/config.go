package dataframes

//Mode is used for enumerating the ilm mode.
type Mode uint8

const (
	//ModeAuto enum 'auto'
	ModeAuto Mode = iota

	//ModeEnabled enum 'true'
	ModeEnabled

	//ModeDisabled enum 'false'
	ModeDisabled
)

// Config is used for unpacking a common.Config.
type Config struct {
	Mode     Mode   `config:"enabled"`
	Source   string `config:"source"`
	Dest     string `config:"dest"`
	Interval string `config:"interval"`

	// CheckExists can disable the check for an existing policy. Check required
	// read_ilm privileges.  If check is disabled the policy will only be
	// installed if Overwrite is enabled.
	CheckExists bool `config:"check_exists"`

	// Enable always overwrite policy mode. This required manage_ilm privileges.
	Overwrite bool `config:"overwrite"`
}
