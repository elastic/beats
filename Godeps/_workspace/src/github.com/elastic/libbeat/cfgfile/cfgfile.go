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

func Read(out interface{}) error {
	filecontent, err := ioutil.ReadFile(*configfile)
	if err != nil {
		return fmt.Errorf("Fail to read %s: %v. Exiting.", *configfile, err)
	}
	if err = yaml.Unmarshal(filecontent, out); err != nil {
		fmt.Errorf("YAML config parsing failed on %s: %v. Exiting.", *configfile, err)
	}

	return nil
}

func IsTestConfig() bool {
	return *testConfig
}
