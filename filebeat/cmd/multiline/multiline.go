package multiline

import (
	"fmt"
	"io"
	"log"

	"github.com/spf13/cobra"
)

var Command *cobra.Command

const name = "multiline"

func init() {
	Command = &cobra.Command{
		Use:   name,
		Short: "Multiline tester",
		Run: func(_ *cobra.Command, args []string) {
			err := run(args)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	addFlags(Command)
}

func run(args []string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	in, err := createReader(args)
	if err != nil {
		return err
	}
	defer in.Close()

	reader, err := createPipeline(config, in)
	if err != nil {
		return fmt.Errorf("creating reader pipeline failed: %v", err)
	}

	for {
		msg, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		fmt.Printf("%s\n", msg.Content)
	}
}
