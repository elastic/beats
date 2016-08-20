package beater

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/elastic/beats/libbeat/beat"

	"github.com/elastic/beats/packetbeat/sniffer"
)

func init() {
	printDevices := flag.Bool("devices", false, "Print the list of devices and exit")

	beat.AddFlagsCallback(func(_ *beat.Beat) error {
		if *printDevices == false {
			return nil
		}

		devs, err := sniffer.ListDeviceNames(true, true)
		if err != nil {
			return fmt.Errorf("Error getting devices list: %v\n", err)
		}
		if len(devs) == 0 {
			fmt.Printf("No devices found.")
			if runtime.GOOS != "windows" {
				fmt.Printf(" You might need sudo?\n")
			} else {
				fmt.Printf("\n")
			}
		}

		for i, dev := range devs {
			fmt.Printf("%d: %s\n", i, dev)
		}
		return beat.GracefulExit
	})
}
