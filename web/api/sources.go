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
	httpClient := &http.Client{
		Timeout: time.Second * 1,
	}

	return func(context echo.Context) error {
		response := GetSourcesResponse{}

		if len(context.QueryParam("node")) > 0 {
			addr := context.QueryParam("node")

			resp, err := httpClient.Get(fmt.Sprintf("http://%s/api/sources", addr))

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
