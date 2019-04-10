package api

import (
	"code.cloudfoundry.org/bytefmt"
	"fmt"
	"github.com/labstack/echo"
	"github.com/pm-connect/log-shipper/monitoring"
	"net/http"
)

type GetTargetsResponse struct {
	Targets []Target
}

type Target struct {
	ID               string
	Connection       string
	State            string
	BytesProcessed   string
	InboundMessages  uint64
	OutboundMessages uint64
	InflightMessages uint64
	DroppedMessages  uint64
	ResentMessages   uint64
	RateLimiters     []RateLimiter
}

type RateLimiter struct {
	ID              string
	Limit           string
	Average         string
	Current         string
	AverageBreached bool
	CurrentBreached bool
	StoredMetrics   int
}

func GetTargetsRoute(monitor *monitoring.Monitor) echo.HandlerFunc {
	return func(context echo.Context) error {
		response := GetTargetsResponse{}

		var targets []Target

		for _, c := range monitor.ConnectionStore.Connections {
			if c.Type != "target" {
				continue
			}

			target := Target{
				ID:               c.Name,
				Connection:       fmt.Sprintf("%s:%d", c.Details.Host, c.Details.Port),
				State:            c.State,
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
