package accrual

import (
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/config"
)

const (
	defaultWorkerCount            int = 10
	defaultResultBufferSize       int = 20
	defaultHarvestBatchSize       int = 100
	defaultQueueHungerThreshold   int = 20
	defaultHarvestIntervalMs      int = 1000
	defaultHarvestSleepIntervalMs int = 100
	defaultWorkerSleepIntervalMs  int = 500
)

type configuration struct {
	ServiceURL string

	HarvestBatchSize     uint64
	ResultBufferSize     uint64
	WorkerCount          int
	QueueHungerThreshold int

	HarvestInterval      time.Duration
	HarvestSleepInterval time.Duration
	WorkerSleepInterval  time.Duration
}

func newConfiguration(cfg *config.AccrualIntegrationConfig) *configuration {
	return &configuration{
		ServiceURL: cfg.AccrualServiceURL,

		WorkerCount:          getOrDefault(cfg.AccrualWorkerCount, defaultWorkerCount),
		ResultBufferSize:     uint64(getOrDefault(cfg.AccrualResultBufferSize, defaultResultBufferSize)),
		HarvestBatchSize:     uint64(getOrDefault(cfg.AccrualHarvestBatchSize, defaultHarvestBatchSize)),
		QueueHungerThreshold: getOrDefault(cfg.AccrualHungerThreshold, defaultQueueHungerThreshold),

		HarvestInterval: time.Duration(
			getOrDefault(cfg.AccrualHarvestIntervalMs, defaultHarvestIntervalMs)) * time.Millisecond,
		HarvestSleepInterval: time.Duration(
			getOrDefault(cfg.AccrualHarvestPauseMs, defaultHarvestSleepIntervalMs)) * time.Millisecond,
		WorkerSleepInterval: time.Duration(
			getOrDefault(cfg.AccrualWorkerSleepIntervalMs, defaultWorkerSleepIntervalMs)) * time.Millisecond,
	}
}

func getOrDefault(value int, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}
