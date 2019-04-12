package api

import (
	"code.cloudfoundry.org/bytefmt"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo"
	"github.com/pm-connect/log-shipper/monitoring"
	"io/ioutil"
	"net/http"
	"time"
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
	httpClient := &http.Client{
		Timeout: time.Second * 1,
	}

	return func(context echo.Context) error {
		response := GetWorkersResponse{}

		if len(context.QueryParam("node")) > 0 {
			addr := context.QueryParam("node")

			resp, err := httpClient.Get(fmt.Sprintf("http://%s/api/workers", addr))

			if err != nil {
				return context.JSON(http.StatusInternalServerError, err)
			}

			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)

			if err != nil {
				return context.JSON(http.StatusInternalServerError, err)
			}

			err = json.Unmarshal(body, &response)

			if err != nil {
				return context.JSON(http.StatusInternalServerError, err)
			}

			return context.JSON(http.StatusOK, response)
		}

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
