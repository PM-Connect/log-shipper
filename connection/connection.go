package connection

import (
	"fmt"
	"net"
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

func OpenTCPConnection(host string, port int) (*net.TCPConn, error) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host,port))
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	return conn, err
}

func StartTCPServer(host string, port int) (*net.TCPListener, error) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	return net.ListenTCP("tcp", tcpAddr)
}
