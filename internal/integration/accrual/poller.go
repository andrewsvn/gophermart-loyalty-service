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
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type poller struct {
	cfg *configuration

	pQueue  *pollingQueue
	results chan<- *pollingResult

	logger *zap.SugaredLogger
}

func newPoller(
	cfg *configuration,
	queue *pollingQueue,
	results chan<- *pollingResult,
	logger *zap.SugaredLogger,
) *poller {
	return &poller{
		cfg:     cfg,
		pQueue:  queue,
		results: results,
		logger:  logger,
	}
}

func (p *poller) start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(p.cfg.WorkerCount)
	for i := 0; i < p.cfg.WorkerCount; i++ {
		go p.pollFunc(ctx, wg, i)
	}
}

func (p *poller) pollFunc(ctx context.Context, wg *sync.WaitGroup, id int) {
	p.logger.Infow("polling worker started", "id", id)
	defer p.logger.Infow("polling worker finished", "id", id)
	defer func() {
		_ = p.logger.Sync()
	}()
	defer wg.Done()

	client := resty.New()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if p.pQueue.isWaiting() {
			p.logger.Debugw("polling is suspended, waiting", "id", id)
			time.Sleep(p.cfg.WorkerSleepInterval)
			continue
		}

		orderID := p.pQueue.getNextOrderID()
		if orderID == "" {
			time.Sleep(p.cfg.WorkerSleepInterval)
			continue
		}

		p.results <- p.updateAccrual(client, orderID)
	}
}

func (p *poller) updateAccrual(client *resty.Client, orderID string) *pollingResult {
	url := fmt.Sprintf("%s/api/orders/%s", p.cfg.ServiceURL, orderID)
	p.logger.Debugw("sending accrual request", "url", url)

	resp, err := client.R().Get(url)
	if err != nil {
		p.logger.Errorw("failed to send accrual request", "error", err)
		return newFailurePollingResult(orderID, outcomeServiceError)
	}

	p.logger.Debugw("accrual response received",
		"statusCode", resp.StatusCode(),
		"contentSize", len(resp.Body()))

	switch resp.StatusCode() {
	case http.StatusOK:
		accrual := &model.OrderAccrual{}
		err := json.Unmarshal(resp.Body(), &accrual)
		if err != nil {
			p.logger.Errorw("failed to unmarshal accrual response", "error", err)
			return newFailurePollingResult(orderID, outcomeServiceError)
		}
		return newSuccessPollingResult(orderID, accrual)
	case http.StatusNoContent:
		return newFailurePollingResult(orderID, outcomeNoData)
	case http.StatusTooManyRequests:
		retryAfterStr := resp.Header().Get("Retry-After")
		retryAfter, err := strconv.ParseInt(retryAfterStr, 10, 64)
		if err != nil {
			p.logger.Errorw("failed to parse Retry-After header from accrual response")
			return newFailurePollingResult(orderID, outcomeServiceError)
		}
		return newRetryLaterPollingResult(orderID, retryAfter)
	}

	p.logger.Errorw("error processing accrual response: unexpected status code",
		"order_id", orderID,
		"status", resp.StatusCode())
	return newFailurePollingResult(orderID, outcomeServiceError)
}
