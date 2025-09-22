package accrual

import (
	"context"
	"sync"

	"github.com/andrewsvn/gophermart-ls/internal/config"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"go.uber.org/zap"
)

type IntegrationFlow struct {
	cfg *configuration

	queue      *pollingQueue
	results    chan *pollingResult
	harvester  *harvester
	poller     *poller
	resultProc *resultProcessor

	logger *zap.SugaredLogger

	prodCancel context.CancelFunc
	prodWG     *sync.WaitGroup
	consWG     *sync.WaitGroup
}

func NewIntegrationFlow(
	intCfg *config.AccrualIntegrationConfig,
	ls repository.LoyaltyStorage,
	l *zap.Logger,
) *IntegrationFlow {
	logger := logging.ComponentLogger(l, "accrual-integration")
	cfg := newConfiguration(intCfg)
	results := make(chan *pollingResult, cfg.ResultBufferSize)

	queue := newPollingQueue(cfg, ls, logger)

	return &IntegrationFlow{
		cfg: cfg,

		queue:      queue,
		results:    results,
		harvester:  newHarvester(cfg, ls, queue, logger),
		poller:     newPoller(cfg, queue, results, logger),
		resultProc: newResultProcessor(cfg, ls, queue, results, logger),

		logger: logger,
	}
}

func (flow *IntegrationFlow) Start() {
	produceCtx, cancel := context.WithCancel(context.Background())
	flow.prodCancel = cancel
	// if pending statuses on orders weren't cleaned up before, do this now
	flow.queue.cleanupPendingStatuses(produceCtx)

	flow.prodWG = &sync.WaitGroup{}
	// accrual result processing
	flow.prodWG.Add(1)
	go flow.harvester.loop(produceCtx, flow.prodWG)

	// polling pQueue
	flow.poller.start(produceCtx, flow.prodWG)

	// separate wait group for channel consumer to make sure it will stop after all others
	flow.consWG = &sync.WaitGroup{}
	flow.consWG.Add(1)
	go flow.resultProc.loop(context.Background(), flow.consWG)

	flow.logger.Infow("accrual integration flow started")
}

func (flow *IntegrationFlow) Shutdown() {
	flow.logger.Infow("accrual integration flow shutting down")

	flow.prodCancel()
	flow.prodWG.Wait()

	close(flow.results)
	flow.consWG.Wait()

	flow.queue.cleanupPendingStatuses(context.Background())
	flow.logger.Infow("accrual integration flow shut down successfully")
}
