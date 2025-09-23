package config

import (
	"flag"
	"fmt"
)

type AccrualIntegrationConfig struct {
	AccrualServiceURL string `env:"ACCRUAL_SYSTEM_ADDRESS"`

	AccrualWorkerCount           int `env:"ACCRUAL_ACCRUAL_WORKER_COUNT"`
	AccrualResultBufferSize      int `env:"ACCRUAL_ACCRUAL_RESULT_BUFFER_SIZE"`
	AccrualHarvestBatchSize      int `env:"ACCRUAL_ACCRUAL_HARVEST_BATCH_SIZE"`
	AccrualHungerThreshold       int `env:"ACCRUAL_ACCRUAL_HUNGER_THRESHOLD"`
	AccrualHarvestIntervalMs     int `env:"ACCRUAL_ACCRUAL_HARVEST_INTERVAL_MS"`
	AccrualHarvestPauseMs        int `env:"ACCRUAL_ACCRUAL_HARVEST_PAUSE_MS"`
	AccrualWorkerSleepIntervalMs int `env:"ACCRUAL_ACCRUAL_WORKER_SLEEP_INTERVAL_MS"`
}

func (cfg *AccrualIntegrationConfig) BindFlags() {
	flag.StringVar(&cfg.AccrualServiceURL, "r", cfg.AccrualServiceURL,
		"The address of the accrual service")
}

func (cfg *AccrualIntegrationConfig) Validate() error {
	if cfg.AccrualServiceURL == "" {
		return fmt.Errorf("accrual service url is required")
	}
	return nil
}
