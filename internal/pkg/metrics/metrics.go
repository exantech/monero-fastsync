package metrics

import (
	"github.com/marpaia/graphite-golang"

	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
)

var g *graphite.Graphite

func Init(c *utils.GraphiteSettings) error {
	if c == nil {
		return nil
	}

	var err error
	g, err = graphite.GraphiteFactory("tcp", c.Host, c.Port, "")

	return err
}

func SimpleSend(stat string, value string) {
	if g == nil {
		return
	}

	if err := g.SimpleSend(stat, value); err != nil {
		logging.Log.Debugf("Failed to send metrics: %s", err.Error())
	}
}
