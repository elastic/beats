package export

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common/cli"
)

// GenExportConfigCmd write to stdout the current configuration in the YAML format.
func GenExportConfigCmd(name, idxPrefix, beatVersion string) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Export current config to stdout",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			return exportConfig(name, idxPrefix, beatVersion)
		}),
	}
}

func exportConfig(name, idxPrefix, beatVersion string) error {
	b, err := instance.NewBeat(name, idxPrefix, beatVersion)
	if err != nil {
		return fmt.Errorf("error initializing beat: %s", err)
	}

	err = b.Init()
	if err != nil {
		return fmt.Errorf("error initializing beat: %s", err)
	}

	var config map[string]interface{}
	err = b.RawConfig.Unpack(&config)
	if err != nil {
		return fmt.Errorf("error unpacking config, error: %s", err)
	}
	res, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("Error converting config to YAML format, error: %s", err)
	}

	os.Stdout.Write(res)
	return nil
}
