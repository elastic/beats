// Copyright (c) 2012 VMware, Inc.

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/elastic/gosigar"
)

func main() {
	concreteSigar := gosigar.ConcreteSigar{}

	uptime := gosigar.Uptime{}
	uptime.Get()
	avg, err := concreteSigar.GetLoadAverage()
	if err != nil {
		fmt.Printf("Failed to get load average")
		return
	}

	fmt.Fprintf(os.Stdout, " %s up %s load average: %.2f, %.2f, %.2f\n",
		time.Now().Format("15:04:05"),
		uptime.Format(),
		avg.One, avg.Five, avg.Fifteen)
}
