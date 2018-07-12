package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common/cli"
	"github.com/elastic/beats/libbeat/management"
)

func getBeat(name, version string) (*instance.Beat, error) {
	b, err := instance.NewBeat(name, "", version)

	if err != nil {
		return nil, fmt.Errorf("error initializing beat: %s", err)
	}

	if err = b.Init(); err != nil {
		return nil, fmt.Errorf("error initializing beat: %s", err)
	}

	return b, nil
}

func genEnrollCmd(name, version string) *cobra.Command {
	keystoreCmd := cobra.Command{
		Use:   "enroll <kibana_url> <enrollment_token>",
		Short: "Enroll in Kibana for Central Management",
		Args:  cobra.ExactArgs(2),
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			beat, err := getBeat(name, version)
			kibanaURL := args[0]
			enrollmentToken := args[1]
			if err != nil {
				return err
			}

			if err = management.Enroll(beat, kibanaURL, enrollmentToken); err != nil {
				return errors.Wrap(err, "Error while enrolling")
			}

			fmt.Println("Enrolled and ready to retrieve settings from Kibana")
			return nil
		}),
	}

	return &keystoreCmd
}
