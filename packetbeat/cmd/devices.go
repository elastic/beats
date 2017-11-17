package cmd

import (
	"fmt"
	"log"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/packetbeat/sniffer"
)

func genDevicesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "devices",
		Short: "List available devices",
		Run: func(cmd *cobra.Command, args []string) {
			printDevicesList()
		},
	}
}

func printDevicesList() {
	lst, err := sniffer.ListDeviceNames(true, true)
	if err != nil {
		log.Fatalf("Error getting devices list: %v\n", err)
	}

	if len(lst) == 0 {
		fmt.Printf("No devices found.")
		if runtime.GOOS != "windows" {
			fmt.Println(" You might need sudo?")
		} else {
			fmt.Println("")
		}
	}

	for i, d := range lst {
		fmt.Printf("%d: %s\n", i, d)
	}
}
