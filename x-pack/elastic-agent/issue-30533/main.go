package main

import (
	"log"
	"os/exec"
	"time"
)

func main() {
	var count, failures int

	tick := time.NewTicker(4 * time.Minute)
	for {
		select {
		case <-tick.C:
			count++
			err := exec.Command("systemctl", "restart", "elastic-agent").Run()
			if err != nil {
				failures++
				log.Printf("[ERROR] %v", err)
			}
		}
		if count%10 == 0 {
			log.Printf("[INFO] count: %d, failures: %d, successes: %d",
				count, failures, count-failures)
		}
	}
}
