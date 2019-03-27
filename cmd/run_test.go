package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/phayes/freeport"
	"github.com/pm-connect/log-shipper/broker"
	"github.com/pm-connect/log-shipper/config"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/message"
	"github.com/pm-connect/log-shipper/protocol"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestRunCommand_StartWithSingleSourceAndSingleTarget(t *testing.T) {
	var data = `
sources:
  testSource:
    provider: test
    targets:
      - testTarget

targets:
  testTarget:
    provider: test
`
	c := config.NewConfig()

	err := c.LoadYAML(data)

	assert.Nil(t, err)

	runCommand := NewRunCommand()

	sourceManager := connection.NewManager()
	targetManager := connection.NewManager()

	logBroker := broker.NewBroker(runCommand.Workers)

	var wg sync.WaitGroup

	wg.Add(2)

	testSource := TestSource{}
	testTarget := TestTarget{}

	sourceManager.AddConnection("testSource", &testSource)
	targetManager.AddConnection("testTarget", &testTarget)

	go func() {
		time.Sleep(time.Until(time.Now().Add(1 * time.Second)))

		logBroker.Stop()
	}()

	err = runCommand.startProcesses(c, sourceManager, targetManager, logBroker)

	assert.Nil(t, err)

	assert.NotEmpty(t, testTarget.ReceivedLogs)

	assert.Equal(t, testSource.SentLogs, len(testTarget.ReceivedLogs))

	for _, l := range testTarget.ReceivedLogs {
		assert.Equal(t, "1", l.SourceMessage.ID)
		assert.Equal(t, "testTarget", l.Target)
		assert.Equal(t, "testSource", l.Source)
		assert.Equal(t, "Test", l.SourceMessage.Message)
		assert.Equal(t, map[string]string{"some-key": "some-value"}, l.SourceMessage.Meta)
	}
}

func TestRunCommand_StartMultipleSourcesAndSingleTarget(t *testing.T) {
	var data = `
sources:
  testSource1:
    provider: test
    targets:
      - testTarget
  testSource2:
    provider: test
    targets:
      - testTarget

targets:
  testTarget:
    provider: test
`
	c := config.NewConfig()

	err := c.LoadYAML(data)

	assert.Nil(t, err)

	runCommand := NewRunCommand()

	sourceManager := connection.NewManager()
	targetManager := connection.NewManager()

	logBroker := broker.NewBroker(runCommand.Workers)

	var wg sync.WaitGroup

	wg.Add(2)

	testSource1 := TestSource{}
	testSource2 := TestSource{}
	testTarget := TestTarget{}

	sourceManager.AddConnection("testSource1", &testSource1)
	sourceManager.AddConnection("testSource2", &testSource2)
	targetManager.AddConnection("testTarget", &testTarget)

	go func() {
		time.Sleep(time.Until(time.Now().Add(1 * time.Second)))

		logBroker.Stop()
	}()

	err = runCommand.startProcesses(c, sourceManager, targetManager, logBroker)

	assert.Nil(t, err)

	assert.NotEmpty(t, testTarget.ReceivedLogs)

	assert.Equal(t, testSource1.SentLogs + testSource2.SentLogs, len(testTarget.ReceivedLogs))

	for _, l := range testTarget.ReceivedLogs {
		assert.Equal(t, "1", l.SourceMessage.ID)
		assert.Equal(t, "testTarget", l.Target)
		assert.Equal(t, "Test", l.SourceMessage.Message)
		assert.Equal(t, map[string]string{"some-key": "some-value"}, l.SourceMessage.Meta)
	}
}

func TestRunCommand_StartSingleSourceAndMultipleTargets(t *testing.T) {
	var data = `
sources:
  testSource:
    provider: test
    targets:
      - testTarget1
      - testTarget2

targets:
  testTarget1:
    provider: test
  testTarget2:
    provider: test
`
	c := config.NewConfig()

	err := c.LoadYAML(data)

	assert.Nil(t, err)

	runCommand := NewRunCommand()

	sourceManager := connection.NewManager()
	targetManager := connection.NewManager()

	logBroker := broker.NewBroker(runCommand.Workers)

	var wg sync.WaitGroup

	wg.Add(2)

	testSource := TestSource{}
	testTarget1 := TestTarget{}
	testTarget2 := TestTarget{}

	sourceManager.AddConnection("testSource", &testSource)
	targetManager.AddConnection("testTarget1", &testTarget1)
	targetManager.AddConnection("testTarget2", &testTarget2)

	go func() {
		time.Sleep(time.Until(time.Now().Add(1 * time.Second)))

		logBroker.Stop()
	}()

	err = runCommand.startProcesses(c, sourceManager, targetManager, logBroker)

	assert.Nil(t, err)

	assert.NotEmpty(t, testTarget1.ReceivedLogs)
	assert.NotEmpty(t, testTarget2.ReceivedLogs)

	assert.Equal(t, testSource.SentLogs, len(testTarget1.ReceivedLogs))
	assert.Equal(t, testSource.SentLogs, len(testTarget2.ReceivedLogs))

	for _, l := range testTarget1.ReceivedLogs {
		assert.Equal(t, "1", l.SourceMessage.ID)
		assert.Equal(t, "testTarget1", l.Target)
		assert.Equal(t, "testSource", l.Source)
		assert.Equal(t, "Test", l.SourceMessage.Message)
		assert.Equal(t, map[string]string{"some-key": "some-value"}, l.SourceMessage.Meta)
	}

	for _, l := range testTarget2.ReceivedLogs {
		assert.Equal(t, "1", l.SourceMessage.ID)
		assert.Equal(t, "testTarget2", l.Target)
		assert.Equal(t, "testSource", l.Source)
		assert.Equal(t, "Test", l.SourceMessage.Message)
		assert.Equal(t, map[string]string{"some-key": "some-value"}, l.SourceMessage.Meta)
	}
}

func TestRunCommand_StartMultipleSourcesAndMultipleTargets(t *testing.T) {
	var data = `
sources:
  testSource1:
    provider: test
    targets:
      - testTarget1
  testSource2:
    provider: test
    targets:
      - testTarget2

targets:
  testTarget1:
    provider: test
  testTarget2:
    provider: test
`
	c := config.NewConfig()

	err := c.LoadYAML(data)

	assert.Nil(t, err)

	runCommand := NewRunCommand()

	sourceManager := connection.NewManager()
	targetManager := connection.NewManager()

	logBroker := broker.NewBroker(runCommand.Workers)

	var wg sync.WaitGroup

	wg.Add(2)

	testSource1 := TestSource{}
	testSource2 := TestSource{}
	testTarget1 := TestTarget{}
	testTarget2 := TestTarget{}

	sourceManager.AddConnection("testSource1", &testSource1)
	sourceManager.AddConnection("testSource2", &testSource2)
	targetManager.AddConnection("testTarget1", &testTarget1)
	targetManager.AddConnection("testTarget2", &testTarget2)

	go func() {
		time.Sleep(time.Until(time.Now().Add(1 * time.Second)))

		logBroker.Stop()
	}()

	err = runCommand.startProcesses(c, sourceManager, targetManager, logBroker)

	assert.Nil(t, err)

	assert.NotEmpty(t, testTarget1.ReceivedLogs)
	assert.NotEmpty(t, testTarget2.ReceivedLogs)

	assert.Equal(t, testSource1.SentLogs, len(testTarget1.ReceivedLogs))
	assert.Equal(t, testSource2.SentLogs, len(testTarget2.ReceivedLogs))

	for _, l := range testTarget1.ReceivedLogs {
		assert.Equal(t, "1", l.SourceMessage.ID)
		assert.Equal(t, "testTarget1", l.Target)
		assert.Equal(t, "testSource1", l.Source)
		assert.Equal(t, "Test", l.SourceMessage.Message)
		assert.Equal(t, map[string]string{"some-key": "some-value"}, l.SourceMessage.Meta)
	}

	for _, l := range testTarget2.ReceivedLogs {
		assert.Equal(t, "1", l.SourceMessage.ID)
		assert.Equal(t, "testTarget2", l.Target)
		assert.Equal(t, "testSource2", l.Source)
		assert.Equal(t, "Test", l.SourceMessage.Message)
		assert.Equal(t, map[string]string{"some-key": "some-value"}, l.SourceMessage.Meta)
	}
}

type TestSource struct {
	SentLogs    int
}
type TestTarget struct {
	ReceivedLogs []*message.TargetMessage
}

func (s *TestSource) Start() (*connection.Details, error) {
	port, err := freeport.GetFreePort()

	if err != nil {
		panic(err)
	}

	ln, err := connection.StartTCPServer("127.0.0.1", port)

	if err != nil {
		panic(err)
	}

	go func() {
		conn, err := ln.AcceptTCP()

		if err != nil {
			panic(err)
		}

		defer conn.Close()

		ready := protocol.WaitForHello(conn)

		if !ready {
			panic(fmt.Errorf("failed to receive HELLO from broker"))
		}

		for {
			log := message.SourceMessage{
				ID:      "1",
				Message: "Test",
				Meta: map[string]string{
					"some-key": "some-value",
				},
			}

			data, err := json.Marshal(log)

			if err != nil {
				panic(err)
			}

			_, err = protocol.WriteNewMessage(conn, protocol.CommandSourceMessage, string(data))

			if err != nil {
				panic(err)
			}

			ok := protocol.WaitForOk(conn)

			if !ok {
				return
			}

			s.SentLogs++
		}
	}()

	return &connection.Details{
		Host: "127.0.0.1",
		Port: port,
	}, nil
}

func (t *TestTarget) Start() (*connection.Details, error) {
	port, err := freeport.GetFreePort()

	if err != nil {
		panic(err)
	}

	ln, err := connection.StartTCPServer("127.0.0.1", port)

	if err != nil {
		panic(err)
	}

	go func() {
		conn, err := ln.AcceptTCP()

		if err != nil {
			panic(err)
		}

		defer func() {
			_, _ = protocol.WriteNewMessage(conn, protocol.CommandBye, "")
			_ = conn.Close()
		}()

		ready := protocol.WaitForHello(conn)

		if !ready {
			panic(fmt.Errorf("failed to receive HELLO from broker"))
		}

		receiveChan := make(chan *protocol.Message)
		errorChan := make(chan error)

		go protocol.ReadToChannel(conn, receiveChan, errorChan)

		for {
			select {
			case msg := <-receiveChan:
				switch msg.Command {
				case protocol.CommandTargetLog:
					log := message.TargetMessage{}

					err = json.Unmarshal([]byte(msg.Data), &log)

					if err != nil {
						panic(err)
					}

					t.ReceivedLogs = append(t.ReceivedLogs, &log)

					protocol.SendOk(conn)
				case protocol.CommandBye:
					_ = conn.Close()
					return
				}
			case err := <-errorChan:
				if err != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}()

	return &connection.Details{
		Host: "127.0.0.1",
		Port: port,
	}, nil
}
