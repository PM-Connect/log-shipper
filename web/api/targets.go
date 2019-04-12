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

type GetTargetsResponse struct {
	Targets []Target `json:"targets"`
}

type Target struct {
	ID               string `json:"id"`
	Connection       string `json:"connection"`
	State            string `json:"state"`
	Provider         string `json:"provider"`
	BytesProcessed   string `json:"bytesProcessed"`
	InboundMessages  uint64 `json:"inboundMessages"`
	OutboundMessages uint64 `json:"outboundMessages"`
	InflightMessages uint64 `json:"inflightMessages"`
	DroppedMessages  uint64 `json:"droppedMessages"`
	ResentMessages   uint64 `json:"resentMessages"`
	RateLimiters     []RateLimiter `json:"rateLimiters"`
}

type RateLimiter struct {
	ID              string `json:"id"`
	Limit           string `json:"limit"`
	Average         string `json:"average"`
	Current         string `json:"current"`
	AverageBreached bool `json:"averageBreached"`
	CurrentBreached bool `json:"currentBreached"`
	StoredMetrics   int `json:"storedMetrics"`
	Interval        string `json:"interval"`
}

func GetTargetsRoute(monitor *monitoring.Monitor) echo.HandlerFunc {
	httpClient := &http.Client{
		Timeout: time.Second * 1,
	}

	return func(context echo.Context) error {
		response := GetTargetsResponse{}

		if len(context.QueryParam("node")) > 0 {
			addr := context.QueryParam("node")

			resp, err := httpClient.Get(fmt.Sprintf("http://%s/api/targets", addr))

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

		var targets []Target

		for _, c := range monitor.ConnectionStore.Connections {
			if c.Type != "target" {
				continue
			}

			target := Target{
				ID:               c.Name,
				Connection:       fmt.Sprintf("%s:%d", c.Details.Host, c.Details.Port),
				State:            c.State,
				Provider:         c.Provider,
				BytesProcessed:   bytefmt.ByteSize(c.Stats.GetBytesProcessed()),
				InboundMessages:  c.Stats.GetMessagesInbound(),
				OutboundMessages: c.Stats.GetMessagesOutbound(),
				InflightMessages: c.Stats.GetInFlightMessages(),
				ResentMessages:   c.Stats.GetResentMessages(),
			}

			if len(c.RateLimiters) > 0 {
				for i, r := range c.RateLimiters {
					_, average, averageBreached := r.IsAverageOverLimit()
					_, currentBreached := r.IsOverLimit()

					target.RateLimiters = append(target.RateLimiters, RateLimiter{
						ID:              fmt.Sprintf("limiter_%d", i),
						Limit:           bytefmt.ByteSize(r.Limit),
						Average:         bytefmt.ByteSize(average),
						Current:         bytefmt.ByteSize(r.GetCurrent()),
						AverageBreached: averageBreached,
						CurrentBreached: currentBreached,
						Interval: r.Interval.String(),
						StoredMetrics:   r.Store.Len(),
					})
				}
			}

			targets = append(targets, target)
		}

		response.Targets = targets

		return context.JSON(http.StatusOK, response)
	}
}
