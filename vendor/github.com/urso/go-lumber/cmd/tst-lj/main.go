package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/urso/go-lumber/server"
)

func main() {
	bind := flag.String("bind", ":5044", "[host]:port to listen on")
	v1 := flag.Bool("v1", false, "Enable protocol version v1")
	v2 := flag.Bool("v2", false, "Enable protocol version v2")
	flag.Parse()

	s, err := server.ListenAndServe(*bind,
		server.V1(*v1),
		server.V2(*v2))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("tcp server up")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		_ = s.Close()
	}()

	for batch := range s.ReceiveChan() {
		log.Printf("Received batch of %v events\n", len(batch.Events))
		batch.ACK()
	}
}
