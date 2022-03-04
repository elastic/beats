package main

import (
	"log"
	"os/exec"
	"time"
)

func main() {
	var count, failures int

	interval := 4 * time.Minute
	log.Printf("starting elastic-agent restart every %s minutes", interval)
	tick := time.NewTicker(interval)
	for {
		select {
		case <-tick.C:
			count++
			started := time.Now()
			log.Printf("[INFO] restarting the agent")
			err := exec.Command("systemctl", "restart", "elastic-agent").Run()
			if err != nil {
				failures++
				log.Printf("[ERROR] %v", err)
			}
			log.Printf("[INFO] restart done, took %s. Stats: count: %d, failures: %d, successes: %d",
				time.Now().Sub(started), count, failures, count-failures)
		}
	}
}
