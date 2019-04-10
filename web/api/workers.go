package api

import (
	"code.cloudfoundry.org/bytefmt"
	"github.com/labstack/echo"
	"github.com/pm-connect/log-shipper/monitoring"
	"net/http"
)

type GetWorkersResponse struct {
	Workers []Worker `json:"workers"`
}

type Worker struct {
	ID               string `json:"id"`
	State            string `json:"state"`
	BytesProcessed   string `json:"bytesProcessed"`
	InboundMessages  uint64 `json:"inboundMessages"`
	OutboundMessages uint64 `json:"outboundMessages"`
	InflightMessages uint64 `json:"inflightMessages"`
}

func GetWorkersRoute(monitor *monitoring.Monitor) echo.HandlerFunc {
	return func(context echo.Context) error {
		response := GetWorkersResponse{}

		var workers []Worker

		for _, c := range monitor.ProcessStore.Processes {
			if c.Type != "worker" {
				continue
			}

			worker := Worker{
				ID:               c.Name,
				State:            c.State,
				BytesProcessed:   bytefmt.ByteSize(c.Stats.GetBytesProcessed()),
				InboundMessages:  c.Stats.GetMessagesInbound(),
				OutboundMessages: c.Stats.GetMessagesOutbound(),
				InflightMessages: c.Stats.GetInFlightMessages(),
			}

			workers = append(workers, worker)
		}

		response.Workers = workers

		return context.JSON(http.StatusOK, response)
	}
}
