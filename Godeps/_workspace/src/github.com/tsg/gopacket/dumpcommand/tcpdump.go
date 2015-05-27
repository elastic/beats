// Copyright 2012 Google, Inc. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

// Package dumpcommand implements a run function for pfdump and pcapdump
// with many similar flags/features to tcpdump.  This code is split out seperate
// from data sources (pcap/pfring) so it can be used by both.
package dumpcommand

import (
	"flag"
	"fmt"
	"github.com/tsg/gopacket"
	_ "github.com/tsg/gopacket/layers" // pulls in all layers decoders
	"log"
	"os"
	"time"
)

var print = flag.Bool("print", true, "Print out packets, if false only prints out statistics")
var maxcount = flag.Int("c", -1, "Only grab this many packets, then exit")
var decoder = flag.String("decoder", "Ethernet", "Name of the decoder to use")
var dump = flag.Bool("X", false, "If true, dump very verbose info on each packet")
var statsevery = flag.Int("stats", 1000, "Output statistics every N packets")
var printErrors = flag.Bool("errors", false, "Print out packet dumps of decode errors, useful for checking decoders against live traffic")
var lazy = flag.Bool("lazy", false, "If true, do lazy decoding")

func Run(src gopacket.PacketDataSource) {
	if !flag.Parsed() {
		log.Fatalln("Run called without flags.Parse() being called")
	}
	var dec gopacket.Decoder
	var ok bool
	if dec, ok = gopacket.DecodersByLayerName[*decoder]; !ok {
		log.Fatalln("No decoder named", *decoder)
	}
	source := gopacket.NewPacketSource(src, dec)
	source.Lazy = *lazy
	source.NoCopy = true
	fmt.Fprintln(os.Stderr, "Starting to read packets")
	count := 0
	bytes := int64(0)
	start := time.Now()
	errors := 0
	truncated := 0
	layertypes := map[gopacket.LayerType]int{}
	for packet := range source.Packets() {
		count++
		bytes += int64(len(packet.Data()))
		if *dump {
			fmt.Println(packet.Dump())
		} else if *print {
			fmt.Println(packet)
		}
		if !*lazy || *print || *dump { // if we've already decoded all layers...
			for _, layer := range packet.Layers() {
				layertypes[layer.LayerType()]++
			}
			if packet.Metadata().Truncated {
				truncated++
			}
			if errLayer := packet.ErrorLayer(); errLayer != nil {
				errors++
				if *printErrors {
					fmt.Println("Error:", errLayer.Error())
					fmt.Println("--- Packet ---")
					fmt.Println(packet.Dump())
				}
			}
		}
		done := *maxcount > 0 && count >= *maxcount
		if count%*statsevery == 0 || done {
			fmt.Fprintf(os.Stderr, "Processed %v packets (%v bytes) in %v, %v errors and %v truncated packets\n", count, bytes, time.Since(start), errors, truncated)
			if len(layertypes) > 0 {
				fmt.Fprintf(os.Stderr, "Layer types seen: %+v\n", layertypes)
			}
		}
		if done {
			break
		}
	}
}
