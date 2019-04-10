package api

import (
	"code.cloudfoundry.org/bytefmt"
	"fmt"
	"github.com/labstack/echo"
	"github.com/pm-connect/log-shipper/monitoring"
	"net/http"
)

type GetSourcesResponse struct {
	Sources []Source
}

type Source struct {
	ID               string
	Connection       string
	State            string
	BytesProcessed   string
	InboundMessages  uint64
	OutboundMessages uint64
	InflightMessages uint64
	DroppedMessages  uint64
	ResentMessages   uint64
}

func GetSourcesRoute(monitor *monitoring.Monitor) echo.HandlerFunc {
	return func(context echo.Context) error {
		response := GetSourcesResponse{}

		var sources []Source

		for _, c := range monitor.ConnectionStore.Connections {
			if c.Type != "source" {
				continue
			}

			source := Source{
				ID:               c.Name,
				Connection:       fmt.Sprintf("%s:%d", c.Details.Host, c.Details.Port),
				State:            c.State,
				BytesProcessed:   bytefmt.ByteSize(c.Stats.GetBytesProcessed()),
				InboundMessages:  c.Stats.GetMessagesInbound(),
				OutboundMessages: c.Stats.GetMessagesOutbound(),
				InflightMessages: c.Stats.GetInFlightMessages(),
				ResentMessages:   c.Stats.GetResentMessages(),
			}

			sources = append(sources, source)
		}

		response.Sources = sources

		return context.JSON(http.StatusOK, response)
	}
}
