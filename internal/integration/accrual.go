package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// TODO: move this to configuration?
const (
	accrualWorkerCount          = 10
	accrualResultBufferSize     = 2 * accrualWorkerCount
	accrualQueueHungerThreshold = 2 * accrualWorkerCount
	accrualHarvestBatchSize     = 10 * accrualWorkerCount
	accrualHarvestInterval      = 2 * time.Second
	accrualHarvestSleepInterval = 100 * time.Millisecond
	accrualWorkerSleepInterval  = time.Second
)

type outcome string

const (
	outcomeSuccess      outcome = "success"
	outcomeNoData       outcome = "no_data"
	outcomeRetryLater   outcome = "retry_later"
	outcomeServiceError outcome = "service_error"
)

type accrualPollingResult struct {
	OrderID       string
	Outcome       outcome
	Payload       *model.OrderAccrual
	Timestamp     time.Time
	RetryAfterSec int64
}

func newSuccessPollingResult(orderID string, payload *model.OrderAccrual) *accrualPollingResult {
	return &accrualPollingResult{
		OrderID:       orderID,
		Outcome:       outcomeSuccess,
		Payload:       payload,
		Timestamp:     time.Now(),
		RetryAfterSec: 0,
	}
}

func newRetryLaterPollingResult(orderID string, retryAfter int64) *accrualPollingResult {
	return &accrualPollingResult{
		OrderID:       orderID,
		Outcome:       outcomeRetryLater,
		Payload:       nil,
		Timestamp:     time.Now(),
		RetryAfterSec: retryAfter,
	}
}

func newFailurePollingResult(orderID string, oc outcome) *accrualPollingResult {
	return &accrualPollingResult{
		OrderID:   orderID,
		Outcome:   oc,
		Timestamp: time.Now(),
	}
}

type AccrualPollingQueue struct {
	repoFacade *repository.Facade
	serviceURL string
	logger     *zap.SugaredLogger

	// orderIDs is a list of orderID values fetched from repository to update from accrual system
	// values are stored according to fetch order, so each fetch call appends returned values to the end
	// and accrual integration workers extract values from the beginning
	// we don't use channel to establish a better control with timers
	orderIDs []string
	// ordersMutex is used for each operation modifying orderIDs
	ordersMutex *sync.Mutex

	results chan *accrualPollingResult

	// we use two separate wait groups to make sure that all producers finish writing into results before
	// closing the channel so the consumer can finish its loop

	prodCancel context.CancelFunc
	prodWG     *sync.WaitGroup
	consWG     *sync.WaitGroup

	// WaitUntilTS must be checked by all pollFunc routines to skip accrual polling loops
	// in case this value is not reached
	// it is updated when any routine receives the response containing 'Retry-After' header
	WaitUntilTS atomic.Pointer[time.Time]
}

func NewAccrualPollingQueue(
	rf *repository.Facade,
	url string,
	l *zap.Logger,
) *AccrualPollingQueue {
	queue := &AccrualPollingQueue{
		repoFacade: rf,
		serviceURL: url,
		logger:     logging.ComponentLogger(l, "accrual-polling"),
	}

	return queue
}

func (pq *AccrualPollingQueue) Start() {
	pq.ordersMutex = &sync.Mutex{}
	pq.results = make(chan *accrualPollingResult, accrualResultBufferSize)

	produceCtx, cancel := context.WithCancel(context.Background())
	pq.prodCancel = cancel
	// if pending statuses on orders weren't cleaned up before, do this now
	pq.cleanupPendingStatuses(produceCtx)

	pq.prodWG = &sync.WaitGroup{}
	// accrual result processing
	pq.prodWG.Add(1)
	go pq.orderIDHarvestFunc(produceCtx)

	// pollers
	pq.prodWG.Add(accrualWorkerCount)
	for i := 0; i < accrualWorkerCount; i++ {
		go pq.pollFunc(produceCtx, i)
	}

	// separate wait group for channel consumer to make sure it will stop after all
	pq.consWG = &sync.WaitGroup{}
	pq.consWG.Add(1)
	go pq.accrualResultProcessing(context.Background())

	pq.logger.Infow("accrual queue started")
}

func (pq *AccrualPollingQueue) Shutdown() {
	pq.logger.Infow("accrual queue shutting down")
	pq.prodCancel()
	pq.prodWG.Wait()

	close(pq.results)
	pq.consWG.Wait()

	pq.cleanupPendingStatuses(context.Background())
	pq.logger.Infow("accrual queue shut down successfully")
}

func (pq *AccrualPollingQueue) orderIDHarvestFunc(ctx context.Context) {
	defer pq.prodWG.Done()
	nextHarvestTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if time.Now().Before(nextHarvestTime) {
			time.Sleep(accrualHarvestSleepInterval)
			continue
		}

		if len(pq.orderIDs) >= accrualQueueHungerThreshold {
			time.Sleep(accrualHarvestSleepInterval)
			continue
		}

		pq.logger.Debugw("fetching new batch of orders for update",
			"batchSize", accrualHarvestBatchSize)
		harvested, err := pq.repoFacade.FetchOrderIDsForUpdate(ctx, accrualHarvestBatchSize)
		if err != nil {
			pq.logger.Warnw("failed to fetch order ids for update", "error", err)
			time.Sleep(accrualHarvestSleepInterval)
			continue
		}

		pq.appendOrderIDs(harvested...)
		pq.logger.Debugw("fetching complete",
			"harvested", len(harvested),
			"newQueueSize", len(pq.orderIDs))
		nextHarvestTime = time.Now().Add(accrualHarvestInterval)
	}
}

func (pq *AccrualPollingQueue) pollFunc(ctx context.Context, id int) {
	pq.logger.Infow("polling worker started", "id", id)
	defer pq.logger.Infow("polling worker finished", "id", id)
	defer func() {
		_ = pq.logger.Sync()
	}()
	defer pq.prodWG.Done()

	client := resty.New()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		waitUntilTS := pq.WaitUntilTS.Load()
		if waitUntilTS != nil && time.Now().Before(*waitUntilTS) {
			pq.logger.Debugw("polling is suspended, waiting", "id", id)
			time.Sleep(accrualWorkerSleepInterval)
			continue
		}

		orderID := pq.getNextOrderID()
		if orderID == "" {
			time.Sleep(accrualWorkerSleepInterval)
			continue
		}

		pq.results <- pq.updateAccrual(client, orderID)
	}
}

func (pq *AccrualPollingQueue) accrualResultProcessing(ctx context.Context) {
	defer pq.consWG.Done()

	// channel will be closed by an orchestrating function, so no context clause here
	for result := range pq.results {
		switch result.Outcome {
		case outcomeSuccess:
			pq.logger.Debugw("updating order accrual data",
				"accrual", result.Payload)

			err := pq.repoFacade.UpdateOrderAccrual(ctx, result.Payload, result.Timestamp)
			if err != nil {
				pq.logger.Errorw("failed to update order accrual result in repository",
					"error", err)
				// let's just put the problematic order again into queue
				// to not lose it in case of some temporary DB problem
				pq.appendOrderIDs(result.OrderID)
			}
		case outcomeRetryLater:
			pq.logger.Debugw("order accrual cannot be fetched due to server overload",
				"orderID", result.OrderID,
				"retryAfter", result.RetryAfterSec)

			waitUntilTS := time.Now().Add(time.Duration(result.RetryAfterSec) * time.Second)
			prevWaitTS := pq.WaitUntilTS.Load()
			if prevWaitTS.Before(waitUntilTS) {
				pq.WaitUntilTS.Store(&waitUntilTS)
			}
			pq.appendOrderIDs(result.OrderID)
		default:
			pq.logger.Debugw("order accrual cannot be fetched",
				"orderID", result.OrderID,
				"reason", result.Outcome)
			pq.appendOrderIDs(result.OrderID)
		}
	}
}

func (pq *AccrualPollingQueue) getNextOrderID() string {
	pq.ordersMutex.Lock()
	defer pq.ordersMutex.Unlock()

	if len(pq.orderIDs) == 0 {
		return ""
	}
	orderID := pq.orderIDs[0]
	pq.orderIDs = pq.orderIDs[1:]
	return orderID
}

func (pq *AccrualPollingQueue) updateAccrual(
	client *resty.Client,
	orderID string,
) *accrualPollingResult {
	url := fmt.Sprintf("%s/api/orders/%s", pq.serviceURL, orderID)
	pq.logger.Debugw("sending accrual request", "url", url)

	resp, err := client.R().Get(url)
	if err != nil {
		pq.logger.Errorw("failed to send accrual request", "error", err)
		return newFailurePollingResult(orderID, outcomeServiceError)
	}

	pq.logger.Debugw("accrual response received",
		"statusCode", resp.StatusCode(),
		"contentSize", len(resp.Body()))

	switch resp.StatusCode() {
	case http.StatusOK:
		accrual := &model.OrderAccrual{}
		err := json.Unmarshal(resp.Body(), &accrual)
		if err != nil {
			pq.logger.Errorw("failed to unmarshal accrual response", "error", err)
			return newFailurePollingResult(orderID, outcomeServiceError)
		}
		return newSuccessPollingResult(orderID, accrual)
	case http.StatusNoContent:
		return newFailurePollingResult(orderID, outcomeNoData)
	case http.StatusTooManyRequests:
		retryAfterStr := resp.Header().Get("Retry-After")
		retryAfter, err := strconv.ParseInt(retryAfterStr, 10, 64)
		if err != nil {
			pq.logger.Errorw("failed to parse Retry-After header from accrual response")
			return newFailurePollingResult(orderID, outcomeServiceError)
		}
		return newRetryLaterPollingResult(orderID, retryAfter)
	}

	pq.logger.Errorw("error processing accrual response: unexpected status code",
		"order_id", orderID,
		"status", resp.StatusCode())
	return newFailurePollingResult(orderID, outcomeServiceError)
}

func (pq *AccrualPollingQueue) appendOrderIDs(newOrderIDs ...string) {
	pq.ordersMutex.Lock()
	pq.orderIDs = append(pq.orderIDs, newOrderIDs...)
	pq.ordersMutex.Unlock()
}

func (pq *AccrualPollingQueue) cleanupPendingStatuses(ctx context.Context) {
	err := pq.repoFacade.ResetPendingOrders(ctx)
	if err != nil {
		pq.logger.Errorw("unable to clean pending status for orders",
			"error", err)
	}
}
