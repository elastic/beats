package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/jsonarr", serveJSONArr)
	http.HandleFunc("/jsonobj", serveJSONObj)
	http.HandleFunc("/", serveJSONObj)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func serveJSONArr(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `[{"hello1":"world1"}, {"hello2": "world2"}]`)
}

func serveJSONObj(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"hello":"world"}`)
}
