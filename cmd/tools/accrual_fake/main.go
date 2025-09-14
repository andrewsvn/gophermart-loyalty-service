package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const (
	serverAddress = ":16666"
)

type Order struct {
	OrderID string `json:"order"`
	Status  string `json:"status"`
	Accrual int64  `json:"accrual,omitempty"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	r := chi.NewRouter()
	r.Get("/api/orders/{number}", getOrderHandler)

	log.Println("starting fake accrual server on ", serverAddress)
	return http.ListenAndServe(serverAddress, r)
}

func getOrderHandler(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "number")
	firstDigit := orderID[0]

	order := &Order{
		OrderID: orderID,
	}
	switch firstDigit {
	case '1', '6':
		order.Status = "REGISTERED"
	case '2', '7':
		order.Status = "PROCESSING"
	case '3', '8':
		order.Status = "INVALID"
	default:
		order.Status = "PROCESSED"
		order.Accrual = rand.Int63n(1000) + 1000
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(order)
	if err != nil {
		log.Printf("error writing response: %v", err)
	}
}
