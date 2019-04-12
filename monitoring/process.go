package monitoring

import "github.com/prometheus/client_golang/prometheus"

type Process struct {
	Name string
	Type string

	Stats *Stats
	State string

	LastLog LastLog
}

func NewProcess(name string, processType string) *Process {
	stats := NewStats("process_", prometheus.Labels{
		"name": name,
		"type": processType,
	})
	return &Process{
		Name:  name,
		Type:  processType,
		Stats: stats,
	}
}

func (c *Process) SetState(state string) {
	c.State = state
}
