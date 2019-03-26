package connection

import (
	"fmt"
)

type Connection interface {
	Start() (*Details, error)
}

type Details struct {
	Host string
	Port int
}

type Manager struct {
	Connections map[string]Connection
}

func NewManager() *Manager {
	manager := Manager{}
	manager.Connections = map[string]Connection{}

	return &manager
}

func (m *Manager) AddConnection(name string, conn Connection) {
	m.Connections[name] = conn
}

func (m *Manager) Start() (map[string]*Details, error) {
	details := map[string]*Details{}

	for name, conn := range m.Connections {
		connectionDetails, err := conn.Start()

		if err != nil {
			return nil, fmt.Errorf("error connecting to source: %s", err)
		}

		details[name] = connectionDetails
	}

	return details, nil
}
