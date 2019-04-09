package monitoring

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Monitor struct {
	ConnectionStore ConnectionStore
	ProcessStore
	Logger *log.Logger
}

type ConnectionStore struct {
	sync.Mutex

	Connections []*Connection
}

type ProcessStore struct {
	sync.Mutex

	Processes []*Process
}

type LastLog struct {
	sync.Mutex

	Log *LogItem
}

type LogItem struct {
	Level log.Level
	Message string
}

func NewMonitor(logger *log.Logger) *Monitor {
	return &Monitor{
		ConnectionStore: ConnectionStore{},
		Logger: logger,
	}
}

func (m *Monitor) AddConnection(conn *Connection) {
	m.ConnectionStore.Add(conn)
}

func (m *Monitor) AddProcess(proc *Process) {
	m.ProcessStore.Add(proc)
}

func (m *Monitor) Log(level log.Level, msg string) {
	if m.Logger != nil {
		m.Logger.Log(level, msg)
	}
}

func (m *Monitor) LogForProcess(proc *Process, level log.Level, msg string) {
	m.Log(level, fmt.Sprintf("%s %s: %s", proc.Name, proc.Type, msg))

	proc.LastLog.Lock()
	proc.LastLog.Log = &LogItem{Level: level, Message: msg}
	proc.LastLog.Unlock()
}

func (m *Monitor) LogForConnection(conn *Connection, level log.Level, msg string) {
	m.Log(level, fmt.Sprintf("%s %s: %s", conn.Name, conn.Type, msg))

	conn.LastLog.Lock()
	conn.LastLog.Log = &LogItem{Level: level, Message: msg}
	conn.LastLog.Unlock()
}

func (cs *ConnectionStore) Add(conn *Connection) {
	cs.Lock()
	cs.Connections = append(cs.Connections, conn)
	cs.Unlock()
}

func (ps *ProcessStore) Add(proc *Process) {
	ps.Lock()
	ps.Processes = append(ps.Processes, proc)
	ps.Unlock()
}