package cfgfile

import (
	"flag"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Command line flags
var configfile *string
var testConfig *bool

func CmdLineFlags(flags *flag.FlagSet, name string) {
	configfile = flags.String("c", fmt.Sprintf("/etc/%s/%s.yml", name, name), "Configuration file")
	testConfig = flags.Bool("test", false, "Test configuration and exit.")
}

// Reads config from yaml file into the given interface structure.
// In case the second param path is not set
func Read(out interface{}, path string) error {

	if path == "" {
		path = *configfile
	}

	filecontent, err := ioutil.ReadFile(path)

	if err != nil {
		return fmt.Errorf("Fail to read %s: %v. Exiting.", path, err)
	}
	if err = yaml.Unmarshal(filecontent, out); err != nil {
		return fmt.Errorf("YAML config parsing failed on %s: %v. Exiting.", path, err)
	}

	return nil
}

func IsTestConfig() bool {
	return *testConfig
}
