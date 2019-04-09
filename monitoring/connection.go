package monitoring

import "github.com/pm-connect/log-shipper/connection"

type Connection struct {
	Details *connection.Details

	Name string
	Type string

	Stats Stats
	State string

	LastLog LastLog
}

func NewConnection(details *connection.Details, name string, connectionType string) *Connection {
	return &Connection{
		Details: details,
		Name: name,
		Type: connectionType,
		Stats: Stats{},
	}
}

func (c *Connection) SetState(state string) {
	c.State = state
}