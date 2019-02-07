package metrics

import (
	"fmt"
	"net"
	"time"

	"github.com/cyberdelia/go-metrics-graphite"
	gometrics "github.com/rcrowley/go-metrics"

	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
)

var (
	BlocksCached    gometrics.Meter
	BlocksScanned   gometrics.Meter
	RequestDuration gometrics.Timer
	Rps             gometrics.Meter
)

func Init(c *utils.GraphiteSettings, prefix string) error {
	if c == nil {
		gometrics.UseNilMetrics = true
	}

	BlocksCached = gometrics.NewMeter()
	if err := gometrics.Register("fsd.blocks.cached", BlocksCached); err != nil {
		logging.Log.Errorf("Failed to register metrics: %s", err.Error())
		return err
	}

	BlocksScanned = gometrics.NewMeter()
	if err := gometrics.Register("fsd.blocks.scanned", BlocksScanned); err != nil {
		logging.Log.Errorf("Failed to register metrics: %s", err.Error())
		return err
	}

	RequestDuration = gometrics.NewTimer()
	if err := gometrics.Register("fsd.requests.duration", RequestDuration); err != nil {
		logging.Log.Errorf("Failed to register metrics: %s", err.Error())
		return err
	}

	Rps = gometrics.NewMeter()
	if err := gometrics.Register("fsd.requests.count", Rps); err != nil {
		logging.Log.Errorf("Failed to register metrics: %s", err.Error())
		return err
	}

	if c != nil {
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port))
		if err != nil {
			logging.Log.Errorf("Failed to resolve graphite address: %s", err.Error())
			return err
		}

		go graphite.Graphite(gometrics.DefaultRegistry, time.Second, prefix, addr)
	}

	return nil
}
