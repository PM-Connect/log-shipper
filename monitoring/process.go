package monitoring

type Process struct {
	Name string
	Type string

	Stats Stats
	State string

	LastLog LastLog
}

func NewProcess(name string, processType string) *Process {
	return &Process{
		Name:  name,
		Type:  processType,
		Stats: Stats{},
	}
}

func (c *Process) SetState(state string) {
	c.State = state
}
