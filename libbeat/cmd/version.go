package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common/cli"
	"github.com/elastic/beats/libbeat/version"
)

func genVersionCmd(name, beatVersion string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show current version info",
		Run: cli.RunWith(
			func(_ *cobra.Command, args []string) error {
				beat, err := instance.NewBeat(name, "", beatVersion)
				if err != nil {
					return fmt.Errorf("error initializing beat: %s", err)
				}

				buildTime := "unknown"
				if bt := version.BuildTime(); !bt.IsZero() {
					buildTime = bt.String()
				}
				fmt.Printf("%s version %s (%s), libbeat %s [%s built %s]\n",
					beat.Info.Beat, beat.Info.Version, runtime.GOARCH, version.GetDefaultVersion(),
					version.Commit(), buildTime)
				return nil
			}),
	}
}
