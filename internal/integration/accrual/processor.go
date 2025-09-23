package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type resultProcessor struct {
	cfg     *configuration
	storage repository.LoyaltyStorage
	logger  *zap.SugaredLogger

	queue   *pollingQueue
	results chan *pollingResult
}

func newResultProcessor(
	cfg *configuration,
	ls repository.LoyaltyStorage,
	queue *pollingQueue,
	results chan *pollingResult,
	logger *zap.SugaredLogger,
) *resultProcessor {
	return &resultProcessor{
		cfg:     cfg,
		storage: ls,
		queue:   queue,
		results: results,
		logger:  logger,
	}
}

func (proc *resultProcessor) start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go proc.loop(ctx, wg)
}

func (proc *resultProcessor) loop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer proc.logger.Info("result processing loop finished")
	defer func() {
		_ = proc.logger.Sync()
	}()

	// channel will be closed by an orchestrating function, so no context clause here
	for result := range proc.results {
		switch result.Outcome {
		case outcomeSuccess:
			proc.logger.Debugw("updating order accrual data",
				"accrual", result.Payload)

			err := proc.storage.UpdateOrderAccrual(ctx, result.Payload, result.Timestamp)
			if err != nil {
				proc.logger.Errorw("failed to update order accrual result in repository",
					"error", err)
				// let's just put the problematic order again into pQueue
				// to not lose it in case of some temporary DB problem
				proc.queue.appendOrderIDs(result.OrderID)
			}
		case outcomeRetryLater:
			proc.logger.Debugw("order accrual cannot be fetched due to server overload",
				"orderID", result.OrderID,
				"retryAfter", result.RetryAfterSec)

			waitUntilTS := time.Now().Add(time.Duration(result.RetryAfterSec) * time.Second)
			proc.queue.updateWaitUntilTS(waitUntilTS)
			proc.queue.appendOrderIDs(result.OrderID)
		default:
			proc.logger.Debugw("order accrual cannot be fetched",
				"orderID", result.OrderID,
				"reason", result.Outcome)
			proc.queue.appendOrderIDs(result.OrderID)
		}
	}
}

func (proc *resultProcessor) updateAccrual(
	client *resty.Client,
	orderID string,
) *pollingResult {
	url := fmt.Sprintf("%s/api/orders/%s", proc.cfg.ServiceURL, orderID)
	proc.logger.Debugw("sending accrual request",
		"url", url,
		"orderID", orderID)

	resp, err := client.R().Get(url)
	if err != nil {
		proc.logger.Errorw("failed to send accrual request", "error", err)
		return newFailurePollingResult(orderID, outcomeServiceError)
	}

	proc.logger.Debugw("accrual response received",
		"statusCode", resp.StatusCode(),
		"contentSize", len(resp.Body()))

	switch resp.StatusCode() {
	case http.StatusOK:
		accrual := &model.OrderAccrual{}
		err := json.Unmarshal(resp.Body(), &accrual)
		if err != nil {
			proc.logger.Errorw("failed to unmarshal accrual response", "error", err)
			return newFailurePollingResult(orderID, outcomeServiceError)
		}
		return newSuccessPollingResult(orderID, accrual)
	case http.StatusNoContent:
		return newFailurePollingResult(orderID, outcomeNoData)
	case http.StatusTooManyRequests:
		retryAfterStr := resp.Header().Get("Retry-After")
		retryAfter, err := strconv.ParseInt(retryAfterStr, 10, 64)
		if err != nil {
			proc.logger.Errorw("failed to parse Retry-After header from accrual response")
			return newFailurePollingResult(orderID, outcomeServiceError)
		}
		return newRetryLaterPollingResult(orderID, retryAfter)
	}

	proc.logger.Errorw("error processing accrual response: unexpected status code",
		"order_id", orderID,
		"status", resp.StatusCode())
	return newFailurePollingResult(orderID, outcomeServiceError)
}
