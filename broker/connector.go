package broker

import (
	"bufio"
	"fmt"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/limiter"
	"github.com/pm-connect/log-shipper/message"
	"github.com/pm-connect/log-shipper/monitoring"
	"github.com/pm-connect/log-shipper/protocol"
	log "github.com/sirupsen/logrus"
	"io"
	"sync"
	"sync/atomic"
)

type ConnectorManager struct {
	SourceConnectors map[string]Connector
	TargetConnectors map[string]Connector

	TargetWaitGroup *sync.WaitGroup
	SourceWaitGroup *sync.WaitGroup

	InFlightWaitGroup *sync.WaitGroup

	TargetStop chan interface{}
	SourceStop chan interface{}

	MessagesInFlight uint64

	Monitor *monitoring.Monitor
}

type Connector interface {
	Handle()
	Send(msg []byte) error
}

type SourceConnector struct {
	Name      string
	Source    *Source
	Manager   *ConnectorManager
	WaitGroup *sync.WaitGroup
	Outbound  chan<- []byte
	StopChan  chan interface{}
}

type TargetConnector struct {
	Name      string
	Target    *Target
	Inbound   chan []byte
	Manager   *ConnectorManager
	WaitGroup *sync.WaitGroup
	StopChan  chan interface{}
}

func NewConnectorManager(monitor *monitoring.Monitor) *ConnectorManager {
	return &ConnectorManager{
		TargetConnectors:  map[string]Connector{},
		SourceConnectors:  map[string]Connector{},
		TargetWaitGroup:   &sync.WaitGroup{},
		SourceWaitGroup:   &sync.WaitGroup{},
		TargetStop:        make(chan interface{}),
		SourceStop:        make(chan interface{}),
		InFlightWaitGroup: &sync.WaitGroup{},
		Monitor:           monitor,
	}
}

func (m *ConnectorManager) Start(sources map[string]*Source, targets map[string]*Target) chan []byte {
	for name, target := range targets {
		connector := &TargetConnector{
			Name:      name,
			Target:    target,
			Inbound:   make(chan []byte),
			Manager:   m,
			WaitGroup: m.TargetWaitGroup,
			StopChan:  m.TargetStop,
		}

		m.TargetConnectors[name] = connector

		m.TargetWaitGroup.Add(1)
		go connector.Handle()
	}

	outbound := make(chan []byte)

	for name, source := range sources {
		connector := &SourceConnector{
			Name:      name,
			Source:    source,
			Manager:   m,
			WaitGroup: m.SourceWaitGroup,
			Outbound:  outbound,
			StopChan:  m.SourceStop,
		}

		m.SourceConnectors[name] = connector

		m.SourceWaitGroup.Add(1)
		go connector.Handle()
	}

	return outbound
}

func (m *ConnectorManager) SendToTarget(target string, msg []byte) error {
	if t, ok := m.TargetConnectors[target]; ok {
		return t.Send(msg)
	} else {
		return fmt.Errorf("unable to find target: %s", target)
	}
}

func (m *ConnectorManager) AddMessage(delta int) {
	atomic.AddUint64(&m.MessagesInFlight, uint64(delta))
	m.InFlightWaitGroup.Add(delta)
}

func (m *ConnectorManager) CompleteMessage() {
	atomic.AddUint64(&m.MessagesInFlight, ^uint64(0))
	m.InFlightWaitGroup.Done()
}

func (m *ConnectorManager) WaitForSources() {
	m.SourceWaitGroup.Wait()
}

func (m *ConnectorManager) WaitForTargets() {
	m.TargetWaitGroup.Wait()
}

func (m *ConnectorManager) StopTargets() {
	m.InFlightWaitGroup.Wait()
	close(m.TargetStop)
	m.WaitForTargets()
}

func (m *ConnectorManager) StopSources() {
	close(m.SourceStop)
	m.WaitForSources()
}

func (t *TargetConnector) Handle() {
	defer t.WaitGroup.Done()

	var rateLimiters []*limiter.RateLimiter

	for _, rule := range t.Target.RateLimitRules {
		rateLimiters = append(rateLimiters, rule.RateLimiter)
	}

	monitoringConnection := monitoring.NewConnection(
		t.Target.ConnectionDetails,
		t.Target.Config.Provider,
		t.Name,
		"target",
		rateLimiters,
	)

	monitoringConnection.SetState("starting")

	if t.Manager.Monitor != nil {
		t.Manager.Monitor.AddConnection(monitoringConnection)
	}

	conn, err := connection.OpenTCPConnection(t.Target.ConnectionDetails.Host, t.Target.ConnectionDetails.Port)
	if err != nil {
		monitoringConnection.SetState("dead")
		t.Manager.Monitor.LogForConnection(monitoringConnection, log.FatalLevel, err.Error())
		return
	}

	defer func() {
		protocol.SendBye(conn)
		_ = conn.Close()
	}()

	ready := protocol.SendHello(conn)

	if !ready {
		monitoringConnection.SetState("dead")
		if t.Manager.Monitor != nil {
			t.Manager.Monitor.LogForConnection(monitoringConnection, log.FatalLevel, "did not receive HELLO from server")
		}
	}

	reader := bufio.NewReader(conn)

ReceiveLoop:
	for {
		monitoringConnection.SetState("waiting")
		select {
		case data := <-t.Inbound:
			monitoringConnection.SetState("processing")
			monitoringConnection.Stats.IncrementBytes(uint64(len(data)))
			monitoringConnection.Stats.IncrementMessagesInbound(uint64(1))
			monitoringConnection.Stats.AddInFlightMessage()

			l, err := message.ProtobufToBroker(data)
			if err != nil {
				monitoringConnection.SetState("dead")
				if t.Manager.Monitor != nil {
					t.Manager.Monitor.LogForConnection(monitoringConnection, log.FatalLevel, err.Error())
				}
				monitoringConnection.Stats.IncrementDroppedMessages(1)
				return
			}

			targetMessage := message.BrokerToTarget(t.Name, l)

			rawData, err := message.ToProtobuf(targetMessage)
			if err != nil {
				monitoringConnection.SetState("dead")
				if t.Manager.Monitor != nil {
					t.Manager.Monitor.LogForConnection(monitoringConnection, log.FatalLevel, err.Error())
				}
				monitoringConnection.Stats.IncrementDroppedMessages(1)
				return
			}

			if passed := t.handleRateLimiting(l, monitoringConnection); !passed {
				if t.Manager.Monitor != nil {
					t.Manager.Monitor.LogForConnection(monitoringConnection, log.InfoLevel, "log failed rate limit check, waiting for next message")
				}
				continue ReceiveLoop
			}

			_, err = protocol.WriteNewMessage(conn, protocol.CommandTargetLog, string(rawData))
			if err != nil {
				monitoringConnection.SetState("dead")
				if t.Manager.Monitor != nil {
					t.Manager.Monitor.LogForConnection(monitoringConnection, log.FatalLevel, err.Error())
				}
				monitoringConnection.Stats.IncrementDroppedMessages(1)
				return
			}

			monitoringConnection.Stats.IncrementMessagesOutbound(uint64(1))

			ok := protocol.WaitForOk(reader)
			if !ok {
				t.Manager.CompleteMessage()
				monitoringConnection.SetState("dead")
				if t.Manager.Monitor != nil {
					t.Manager.Monitor.LogForConnection(monitoringConnection, log.FatalLevel, err.Error())
				}
				monitoringConnection.Stats.IncrementDroppedMessages(1)
				return
			}

			for _, rule := range t.Target.RateLimitRules {
				bytes := uint64(len(rawData))
				rule.RateLimiter.Increment(bytes)
			}
			t.Manager.CompleteMessage()
			monitoringConnection.Stats.RemoveInFlightMessage()
		case <-t.StopChan:
			protocol.SendBye(conn)
			_ = conn.Close()
			monitoringConnection.SetState("finished")
			return
		}
	}
}

func (t *TargetConnector) handleRateLimiting(m *message.BrokerMessage, monitoringConnection *monitoring.Connection) bool {
	for _, rule := range t.Target.RateLimitRules {
		if _, _, over := rule.RateLimiter.IsAverageOverLimit(); over {
			// TODO: Send Alerts

			switch rule.BreachBehaviour.Action {
			case "discard":
				t.Manager.CompleteMessage()
				monitoringConnection.Stats.RemoveInFlightMessage()
				monitoringConnection.Stats.IncrementDroppedMessages(uint64(1))
				return false
			case "fallback":
				newTargets := []string{rule.BreachBehaviour.Target}
				m.Targets = newTargets

				rawData, err := message.ToProtobuf(m)
				if err != nil {
					monitoringConnection.SetState("dead")
					if t.Manager.Monitor != nil {
						t.Manager.Monitor.LogForConnection(monitoringConnection, log.ErrorLevel, err.Error())
					}
					return false
				}

				if t.Manager.Monitor != nil {
					t.Manager.Monitor.LogForConnection(monitoringConnection, log.InfoLevel, fmt.Sprintf("rate limit reached, falling back to target \"%s\"", rule.BreachBehaviour.Target))
				}

				err = t.Manager.SendToTarget(rule.BreachBehaviour.Target, rawData)

				monitoringConnection.Stats.IncrementMessagesOutbound(uint64(1))
				monitoringConnection.Stats.IncrementResentMessages(uint64(1))

				if err != nil {
					if t.Manager.Monitor != nil {
						t.Manager.Monitor.LogForConnection(monitoringConnection, log.ErrorLevel, err.Error())
					}
				}

				return false
			}
		}
	}

	return true
}

func (t *TargetConnector) Send(msg []byte) error {
	t.Inbound <- msg
	return nil
}

func (s *SourceConnector) Handle() {
	defer s.WaitGroup.Done()

	monitoringConnection := monitoring.NewConnection(
		s.Source.ConnectionDetails,
		s.Source.Config.Provider,
		s.Name,
		"source",
		nil,
	)

	monitoringConnection.SetState("starting")

	if s.Manager.Monitor != nil {
		s.Manager.Monitor.AddConnection(monitoringConnection)
	}

	conn, err := connection.OpenTCPConnection(s.Source.ConnectionDetails.Host, s.Source.ConnectionDetails.Port)
	if err != nil {
		monitoringConnection.SetState("dead")
		if s.Manager.Monitor != nil {
			s.Manager.Monitor.LogForConnection(monitoringConnection, log.FatalLevel, err.Error())
		}
		return
	}

	ready := protocol.SendHello(conn)
	if !ready {
		monitoringConnection.SetState("dead")
		if s.Manager.Monitor != nil {
			s.Manager.Monitor.LogForConnection(monitoringConnection, log.FatalLevel, "failed to send hello to source")
		}
		return
	}

	receiveChan := make(chan *protocol.Message)
	errorChan := make(chan error)

	reader := bufio.NewReader(conn)

	go protocol.ReadToChannel(reader, receiveChan, errorChan)

	defer func() {
		protocol.SendBye(conn)
		_ = conn.Close()
	}()

	for {
		monitoringConnection.SetState("waiting")
		select {
		case msg := <-receiveChan:
			switch msg.Command {
			case protocol.CommandSourceMessage:
				monitoringConnection.SetState("processing")
				monitoringConnection.Stats.IncrementBytes(uint64(len(msg.Data)))
				monitoringConnection.Stats.IncrementMessagesInbound(uint64(1))
				monitoringConnection.Stats.AddInFlightMessage()

				sourceMsg, err := message.ProtobufToSource(msg.Data)
				if err != nil {
					monitoringConnection.SetState("dead")
					if s.Manager.Monitor != nil {
						s.Manager.Monitor.LogForConnection(monitoringConnection, log.FatalLevel, err.Error())
					}
					return
				}

				l := message.SourceToBroker(s.Name, s.Source.Targets, sourceMsg)

				rawData, err := message.ToProtobuf(l)
				if err != nil {
					monitoringConnection.SetState("dead")
					if s.Manager.Monitor != nil {
						s.Manager.Monitor.LogForConnection(monitoringConnection, log.FatalLevel, err.Error())
					}
					return
				}

				protocol.SendOk(conn)

				s.Manager.AddMessage(len(l.Targets))
				s.Outbound <- rawData
				monitoringConnection.Stats.IncrementMessagesOutbound(uint64(1))
				monitoringConnection.Stats.RemoveInFlightMessage()
			}
		case err := <-errorChan:
			if err != nil && err == io.EOF {
				monitoringConnection.SetState("finished")
				return
			} else if err != nil {
				monitoringConnection.SetState("dead")
				if s.Manager.Monitor != nil {
					s.Manager.Monitor.LogForConnection(monitoringConnection, log.FatalLevel, err.Error())
				}
				return
			}
		case <-s.StopChan:
			_ = conn.Close()
			monitoringConnection.SetState("finished")
			return
		}
	}
}

func (s *SourceConnector) Send(msg []byte) error {
	return fmt.Errorf("not implemented, sources do not receive messages")
}
