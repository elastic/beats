package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strconv"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/pcap"
)

// For a given PCAP file, generates one file per record (only the record data without any extra PCAP info)
func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: prepare_corpus <filename.pcap>")
	}
	pcapFile := os.Args[1]

	handle, err := pcap.OpenOffline(pcapFile)
	fatalIfErr(err)
	defer handle.Close()

	b := make([]byte, 4)
	_, err = rand.Read(b)
	fatalIfErr(err)
	prefix := hex.EncodeToString(b)

	i := 0
	for packet := range gopacket.NewPacketSource(handle, handle.LinkType()).Packets() {
		file, err := os.Create("initial_" + prefix + "_" + strconv.Itoa(i))
		fatalIfErr(err)

		_, err = file.Write(packet.Data())
		fatalIfErr(err)

		err = file.Close()
		fatalIfErr(err)

		i++
	}
}

func fatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
