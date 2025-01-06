package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func TestServer() {
	// a simple go server with port from command line
	port := os.Args[1]
	fmt.Println("Starting server on port", port)
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		// time.Sleep(5 * time.Second)
		log.Println("Received request from", r.RemoteAddr, "on port", port)
		fmt.Fprintf(w, "Hello, World! from port %s \n", port)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Healthy %s \n", port)
	})
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
