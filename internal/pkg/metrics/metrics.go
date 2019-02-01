package metrics

import (
	"github.com/marpaia/graphite-golang"

	"github.com/exantech/monero-fastsync/internal/pkg/utils"
)

var g *graphite.Graphite

func Init(c *utils.GraphiteSettings) error {
	if c == nil {
		g = graphite.NewGraphiteNop("", 0)
		return nil
	}

	var err error
	g, err = graphite.GraphiteFactory("tcp", c.Host, c.Port, "")

	return err
}

func Graphite() *graphite.Graphite {
	return g
}
