package api

import (
	"code.cloudfoundry.org/bytefmt"
	"fmt"
	"github.com/labstack/echo"
	"github.com/pm-connect/log-shipper/monitoring"
	"net/http"
)

type GetSourcesResponse struct {
	Sources []Source `json:"sources"`
}

type Source struct {
	ID               string `json:"id"`
	Connection       string `json:"connection"`
	State            string `json:"state"`
	Provider         string `json:"provider"`
	BytesProcessed   string `json:"bytesProcessed"`
	InboundMessages  uint64 `json:"inboundMessages"`
	OutboundMessages uint64 `json:"outboundMessages"`
	InflightMessages uint64 `json:"inflightMessages"`
	DroppedMessages  uint64 `json:"droppedMessages"`
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
				Provider:         c.Provider,
				BytesProcessed:   bytefmt.ByteSize(c.Stats.GetBytesProcessed()),
				InboundMessages:  c.Stats.GetMessagesInbound(),
				OutboundMessages: c.Stats.GetMessagesOutbound(),
				InflightMessages: c.Stats.GetInFlightMessages(),
			}

			sources = append(sources, source)
		}

		response.Sources = sources

		return context.JSON(http.StatusOK, response)
	}
}
