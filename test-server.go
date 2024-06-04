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
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World! from port %s \n", port)
	})
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
