package api

import (
	"code.cloudfoundry.org/bytefmt"
	"github.com/labstack/echo"
	"github.com/pm-connect/log-shipper/monitoring"
	"net/http"
)

type GetWorkersResponse struct {
	Workers []Worker
}

type Worker struct {
	ID               string
	State            string
	BytesProcessed   string
	InboundMessages  uint64
	OutboundMessages uint64
	InflightMessages uint64
	DroppedMessages  uint64
	ResentMessages   uint64
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
				ResentMessages:   c.Stats.GetResentMessages(),
			}

			workers = append(workers, worker)
		}

		response.Workers = workers

		return context.JSON(http.StatusOK, response)
	}
}
