package main

import (
	"log"
	"net/http"

	"go-template/internal/health"
	"go-template/internal/item"
)

func main() {
	mux := http.NewServeMux()
	health.Register(mux)
	item.Register(mux)

	log.Println("go-template ouvindo em http://127.0.0.1:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", mux))
}
