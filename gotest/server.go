package main

import (
	"fmt"
	"net/http"
	"log"
)

func main() {
	http.HandleFunc("/ping", ping)
	http.HandleFunc("/", hello)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "world")
}
func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "pong")
}
