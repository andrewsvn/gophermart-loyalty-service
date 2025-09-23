package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	maxRequestsBeforeCooldown int64 = 60
	cooldownSeconds           int64 = 10
)

type accrualData struct {
	OrderID string `json:"order"`
	Status  string `json:"status"`
	Accrual int64  `json:"accrual,omitempty"`
}

type fakeService struct {
	requestsSinceLastCooldown int64
	lastCooldownTime          *time.Time
}

func (fs *fakeService) getOrderHandler(w http.ResponseWriter, r *http.Request) {
	if fs.requestsSinceLastCooldown >= maxRequestsBeforeCooldown {
		if fs.lastCooldownTime == nil {
			ts := time.Now()
			fs.lastCooldownTime = &ts
		}

		durationBeforeCooldown := time.Duration(cooldownSeconds)*time.Second - time.Since(*fs.lastCooldownTime)
		secBeforeCooldown := durationBeforeCooldown.Seconds() + 1
		if secBeforeCooldown > 0 {
			w.Header().Set("Retry-After", strconv.FormatInt(cooldownSeconds, 10))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		// reset cooldown
		fs.requestsSinceLastCooldown = 0
		fs.lastCooldownTime = nil
	}

	fs.requestsSinceLastCooldown++

	orderID := chi.URLParam(r, "number")
	data := fs.generateAccrualData(orderID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("error writing response: %v", err)
	}
}

func (fs *fakeService) generateAccrualData(orderID string) *accrualData {
	firstDigit := orderID[0]

	data := &accrualData{
		OrderID: orderID,
	}
	switch firstDigit {
	case '1', '6':
		data.Status = "REGISTERED"
	case '2', '7':
		data.Status = "PROCESSING"
	case '3', '8':
		data.Status = "INVALID"
	default:
		data.Status = "PROCESSED"
		data.Accrual = rand.Int63n(1000) + 1000
	}
	return data
}
