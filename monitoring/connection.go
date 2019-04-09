package monitoring

import (
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/limiter"
)

type Connection struct {
	Details *connection.Details

	Name string
	Type string

	Stats Stats
	State string

	LastLog LastLog

	RateLimiters []*limiter.RateLimiter
}

func NewConnection(details *connection.Details, name string, connectionType string, rateLimiters []*limiter.RateLimiter) *Connection {
	return &Connection{
		Details: details,
		Name:    name,
		Type:    connectionType,
		Stats:   Stats{},
		RateLimiters: rateLimiters,
	}
}

func (c *Connection) SetState(state string) {
	c.State = state
}
