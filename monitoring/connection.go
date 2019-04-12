package monitoring

import (
	"fmt"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/limiter"
	"github.com/prometheus/client_golang/prometheus"
)

type Connection struct {
	Details *connection.Details

	Name string
	Type string
	Provider string

	Stats *Stats
	State string

	LastLog LastLog

	RateLimiters []*limiter.RateLimiter
}

func NewConnection(details *connection.Details, provider string, name string, connectionType string, rateLimiters []*limiter.RateLimiter) *Connection {
	stats := NewStats("connection_", prometheus.Labels{
		"connection": fmt.Sprintf("%s:%d", details.Host, details.Port),
		"name": name,
		"type": connectionType,
		"provider": provider,
	})

	return &Connection{
		Details:      details,
		Provider:     provider,
		Name:         name,
		Type:         connectionType,
		Stats:        stats,
		RateLimiters: rateLimiters,
	}
}

func (c *Connection) SetState(state string) {
	c.State = state
}
