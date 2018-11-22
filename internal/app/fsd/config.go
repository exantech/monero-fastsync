package fsd

import (
	"errors"
	"fmt"
	"time"

	"github.com/exantech/monero-fastsync/internal/pkg/utils"
)

type Config struct {
	LogLevel      string           `yaml:"log_level"`
	Server        string           `yaml:"server"`
	Pprof         string           `yaml:"pprof"`
	BlockchainDb  utils.DbSettings `yaml:"blockchain_db"`
	Network       string           `yaml:"network"`
	Workers       int              `yaml:"workers"`
	ProcessBlocks int              `yaml:"process_blocks"`
	ResultBlocks  int              `yaml:"result_blocks"`
	JobLifetime   time.Duration    `yaml:"job_lifetime"`
}

func MakeDefaultConfig() Config {
	return Config{
		LogLevel:      "info",
		Workers:       10,
		ProcessBlocks: 2000,
		ResultBlocks:  1000,
		JobLifetime:   time.Minute,
	}
}

func (c *Config) Validate() error {
	switch c.Network {
	case "mainnet", "stagenet": // testnet is currently not supported
	default:
		return errors.New(fmt.Sprintf("unknown network: %s", c.Network))
	}

	return nil
}
