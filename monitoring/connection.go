package monitoring

import (
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/limiter"
)

type Connection struct {
	Details *connection.Details

	Name string
	Type string
	Provider string

	Stats Stats
	State string

	LastLog LastLog

	RateLimiters []*limiter.RateLimiter
}

func NewConnection(details *connection.Details, provider string, name string, connectionType string, rateLimiters []*limiter.RateLimiter) *Connection {
	return &Connection{
		Details:      details,
		Provider:     provider,
		Name:         name,
		Type:         connectionType,
		Stats:        Stats{},
		RateLimiters: rateLimiters,
	}
}

func (c *Connection) SetState(state string) {
	c.State = state
}
