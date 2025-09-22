package accrual

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"go.uber.org/zap"
)

type pollingQueue struct {
	cfg *configuration

	storage repository.LoyaltyStorage
	logger  *zap.SugaredLogger

	// orderIDs is a list of orderID values fetched from repository to update from accrual system
	// values are stored according to fetch order, so each fetch call appends returned values to the end
	// and accrual integration workers extract values from the beginning
	// we don't use channel to establish a better control with timers
	orderIDs []string
	// mutex is used for each operation modifying orderIDs
	mutex *sync.Mutex

	// WaitUntilTS must be checked by all pollFunc routines to skip accrual polling loops
	// in case this value is not reached
	// it is updated when any routine receives the response containing 'Retry-After' header
	WaitUntilTS atomic.Pointer[time.Time]
}

func newPollingQueue(
	cfg *configuration,
	ls repository.LoyaltyStorage,
	logger *zap.SugaredLogger,
) *pollingQueue {
	queue := &pollingQueue{
		cfg:     cfg,
		storage: ls,
		logger:  logger,

		orderIDs: make([]string, 0),
		mutex:    &sync.Mutex{},
	}

	return queue
}

func (pq *pollingQueue) getNextOrderID() string {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	if len(pq.orderIDs) == 0 {
		return ""
	}
	orderID := pq.orderIDs[0]
	pq.orderIDs = pq.orderIDs[1:]
	return orderID
}

func (pq *pollingQueue) appendOrderIDs(newOrderIDs ...string) {
	pq.mutex.Lock()
	pq.orderIDs = append(pq.orderIDs, newOrderIDs...)
	pq.mutex.Unlock()
}

func (pq *pollingQueue) isHungry() bool {
	return len(pq.orderIDs) < pq.cfg.QueueHungerThreshold
}

func (pq *pollingQueue) cleanupPendingStatuses(ctx context.Context) {
	err := pq.storage.ResetPendingOrders(ctx)
	if err != nil {
		pq.logger.Errorw("unable to clean pending status for orders",
			"error", err)
	}
}

func (pq *pollingQueue) updateWaitUntilTS(waitUntilTS time.Time) {
	prevWaitTS := pq.WaitUntilTS.Load()
	if prevWaitTS.Before(waitUntilTS) {
		pq.WaitUntilTS.Store(&waitUntilTS)
	}
}

func (pq *pollingQueue) isWaiting() bool {
	waitUntilTS := pq.WaitUntilTS.Load()
	return waitUntilTS != nil && time.Now().Before(*waitUntilTS)
}
