package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", serve)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func serve(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{"hello":"world"}`)
}
