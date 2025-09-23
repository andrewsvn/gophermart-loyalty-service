package accrual

import (
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type pollingQueue struct {
	cfg *configuration

	logger *zap.SugaredLogger

	// orderIDs is a list of orderID values fetched from repository to update from accrual system
	// values are stored according to fetch order, so each fetch call appends returned values to the end
	// and accrual integration workers extract values from the beginning
	// we don't use channel to establish a better control with timers
	orderIDs []string
	// mutex is used for each operation modifying orderIDs
	mutex *sync.Mutex

	// WaitUntilTS must be checked by all polling routines to skip accrual polling loops until this moment
	// it is updated when any polling routine receives a response containing 'Retry-After' header
	WaitUntilTS atomic.Pointer[time.Time]
}

func newPollingQueue(
	cfg *configuration,
	logger *zap.SugaredLogger,
) *pollingQueue {
	queue := &pollingQueue{
		cfg:    cfg,
		logger: logger,

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

	pq.logger.Debugw("added new orderIDs",
		"batchSize", len(newOrderIDs),
		"newQueueSize", len(pq.orderIDs))

	pq.mutex.Unlock()
}

func (pq *pollingQueue) isHungry() bool {
	return len(pq.orderIDs) < pq.cfg.QueueHungerThreshold
}

func (pq *pollingQueue) updateWaitUntilTS(waitUntilTS time.Time) {
	prevWaitTS := pq.WaitUntilTS.Load()
	if prevWaitTS.Before(waitUntilTS) {
		pq.WaitUntilTS.Store(&waitUntilTS)
		pq.logger.Debugw("stored a new timestamp for waiting", "waitUntilTS", waitUntilTS)
	}
}

func (pq *pollingQueue) isWaiting() bool {
	waitUntilTS := pq.WaitUntilTS.Load()
	return waitUntilTS != nil && time.Now().Before(*waitUntilTS)
}
