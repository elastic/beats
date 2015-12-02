package main

import "flag"

func main() {
	cpus := flag.Int("cpus", 1, "Number of burning goroutines to start")
	flag.Parse()

	x := 17
	var c chan bool
	for i := 0; i < *cpus; i++ {
		go func() {
			for {
				x = x * x
			}
		}()
	}
	<-c
}
