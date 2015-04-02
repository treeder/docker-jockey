package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/ping", ping)
	http.HandleFunc("/", hello)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello world!")
}
func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "pong")
}
