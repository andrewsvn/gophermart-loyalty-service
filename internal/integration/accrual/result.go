package accrual

import (
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/model"
)

type outcome string

const (
	outcomeSuccess      outcome = "success"
	outcomeNoData       outcome = "no_data"
	outcomeRetryLater   outcome = "retry_later"
	outcomeServiceError outcome = "service_error"
)

type pollingResult struct {
	OrderID       string
	Outcome       outcome
	Payload       *model.OrderAccrual
	Timestamp     time.Time
	RetryAfterSec int64
}

func newSuccessPollingResult(orderID string, payload *model.OrderAccrual) *pollingResult {
	return &pollingResult{
		OrderID:       orderID,
		Outcome:       outcomeSuccess,
		Payload:       payload,
		Timestamp:     time.Now(),
		RetryAfterSec: 0,
	}
}

func newRetryLaterPollingResult(orderID string, retryAfter int64) *pollingResult {
	return &pollingResult{
		OrderID:       orderID,
		Outcome:       outcomeRetryLater,
		Payload:       nil,
		Timestamp:     time.Now(),
		RetryAfterSec: retryAfter,
	}
}

func newFailurePollingResult(orderID string, oc outcome) *pollingResult {
	return &pollingResult{
		OrderID:   orderID,
		Outcome:   oc,
		Timestamp: time.Now(),
	}
}
