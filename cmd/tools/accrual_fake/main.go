package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const (
	serverAddress = ":16666"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	fake := &fakeService{}

	r := chi.NewRouter()
	r.Get("/api/orders/{number}", fake.getOrderHandler)

	log.Println("starting fake accrual server on ", serverAddress)
	return http.ListenAndServe(serverAddress, r)
}
