package fsd

import (
	"errors"
	"fmt"

	"github.com/exantech/monero-fastsync/internal/pkg/utils"
)

type Config struct {
	LogLevel     string           `yaml:"log_level"`
	Server       string           `yaml:"server"`
	Pprof        string           `yaml:"pprof"`
	BlockchainDb utils.DbSettings `yaml:"blockchain_db"`
	Network      string           `yaml:"network"`
}

func (c *Config) Validate() error {
	switch c.Network {
	case "mainnet", "stagenet": // testnet is currently not supported
	default:
		return errors.New(fmt.Sprintf("unknown network: %s", c.Network))
	}

	return nil
}
