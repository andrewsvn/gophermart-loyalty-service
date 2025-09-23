package accrual

import (
	"context"
	"sync"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"go.uber.org/zap"
)

type harvester struct {
	cfg *configuration

	storage repository.LoyaltyStorage
	queue   *pollingQueue
	logger  *zap.SugaredLogger
}

func newHarvester(
	cfg *configuration,
	ls repository.LoyaltyStorage,
	queue *pollingQueue,
	logger *zap.SugaredLogger,
) *harvester {
	return &harvester{
		cfg:     cfg,
		storage: ls,
		queue:   queue,
		logger:  logger,
	}
}

func (h *harvester) start(ctx context.Context, wg *sync.WaitGroup) {
	h.cleanupPendingStatuses(ctx)

	wg.Add(1)
	go h.loop(ctx, wg)
}

func (h *harvester) loop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer h.logger.Info("order harvesting loop finished")
	defer func() {
		_ = h.logger.Sync()
	}()

	nextHarvestTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if time.Now().Before(nextHarvestTime) {
			time.Sleep(h.cfg.HarvestSleepInterval)
			continue
		}

		if !h.queue.isHungry() {
			h.logger.Debugw("queue is not hungry yet, sleeping")
			time.Sleep(h.cfg.HarvestSleepInterval)
			continue
		}

		h.logger.Debugw("harvesting new batch of orders for update",
			"batchSize", h.cfg.HarvestBatchSize)
		harvested, err := h.storage.FetchOrderIDsForUpdate(ctx, h.cfg.HarvestBatchSize)
		if err != nil {
			h.logger.Warnw("failed to fetch order ids for update", "error", err)
			time.Sleep(h.cfg.HarvestSleepInterval)
			continue
		}

		h.queue.appendOrderIDs(harvested...)
		h.logger.Debugw("harvesting complete", "harvested", len(harvested))
		nextHarvestTime = time.Now().Add(h.cfg.HarvestInterval)
	}
}

func (h *harvester) cleanupPendingStatuses(ctx context.Context) {
	err := h.storage.ResetPendingOrders(ctx)
	if err != nil {
		h.logger.Errorw("unable to clean pending status for orders",
			"error", err)
	}
}
