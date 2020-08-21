package main

// Import inputs, processors and other functionality from legacy beats code
// base All legacy imports from Beats that register with beats specific global
// registries must be included here, and only here.

import (
	// Packetbeat protocol analyzers for use by the sniffer input
	_ "github.com/elastic/beats/v7/packetbeat/include"
)
