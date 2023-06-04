package main

import (
	"chat/internal/handlers"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

var port = 8080

func main() {
	mux := routes()

	fmt.Print("Starting websocket channel listener\n")
	go handlers.ListenToWsChan()

	fmt.Printf("Starting server on port: %d\n", port)

	err := http.ListenAndServe(":"+strconv.Itoa(port), mux)

	if err != nil {
		log.Fatal("failed to start server")
	}
}
