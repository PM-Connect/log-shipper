package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/phayes/freeport"
	"github.com/pm-connect/log-shipper/broker"
	"github.com/pm-connect/log-shipper/config"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/protocol"
	"github.com/stretchr/testify/assert"
	"net"
	"sync"
	"testing"
)

func TestRunStartsSourcesAndTargets(t *testing.T) {
	var data = `
workers: 1

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

	logBroker := broker.NewBroker(c.Workers)

	var wg sync.WaitGroup

	wg.Add(2)

	testSource := TestSource{WaitGroup: &wg}
	testTarget := TestTarget{WaitGroup: &wg}

	sourceManager.AddConnection("testSource", &testSource)
	targetManager.AddConnection("testTarget", &testTarget)

	go func() {
		for testSource.SentLogs < 5 {}

		logBroker.Stop()
	}()

	err = runCommand.startProcesses(c, sourceManager, targetManager, logBroker)

	assert.Nil(t, err)

	assert.NotEmpty(t, testTarget.ReceivedLogs)

	assert.Equal(t, testSource.SentLogs, len(testTarget.ReceivedLogs))
	assert.Equal(t, testSource.SentLogs, logBroker.ProcessedByWorker)
	assert.Equal(t, logBroker.ProcessedByWorker, len(testTarget.ReceivedLogs))
	assert.Equal(t, testSource.SentLogs, logBroker.ReceivedFromSources)
	assert.Equal(t, logBroker.SentToTargets, len(testTarget.ReceivedLogs))

	for _, l := range testTarget.ReceivedLogs {
		assert.Equal(t, "1", l.SourceLog.ID)
		assert.Equal(t, "testTarget", l.Target)
		assert.Equal(t, "testSource", l.Source)
		assert.Equal(t, "Test", l.SourceLog.Message)
		assert.Equal(t, map[string]string{"some-key": "some-value"}, l.SourceLog.Meta)
	}
}

type TestSource struct {
	SentLogs    int
	WaitGroup *sync.WaitGroup
}
type TestTarget struct {
	ReceivedLogs []*broker.TargetLog
	WaitGroup *sync.WaitGroup
}

func (s *TestSource) Start() (*connection.Details, error) {
	port, err := freeport.GetFreePort()

	if err != nil {
		panic(err)
	}

	tcpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	ln, err := net.ListenTCP("tcp", tcpAddr)

	if err != nil {
		panic(err)
	}

	go func() {
		defer s.WaitGroup.Done()

		conn, err := ln.AcceptTCP()

		if err != nil {
			panic(err)
		}

		defer conn.Close()

		message, err := protocol.ReadMessage(conn)

		if err != nil {
			panic(err)
		}

		if message.Command != protocol.CommandHello {
			panic(fmt.Errorf("expected HELLO command from client, received %s", message.Command))
		}

		for {
			log := broker.SourceLog{
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

			_, err = protocol.WriteNewMessage(conn, protocol.CommandSourceLog, string(data))

			if err != nil {
				panic(err)
			}

			response, err := protocol.ReadMessage(conn)

			if err == nil && (response.Command != protocol.CommandOk && response.Command != protocol.CommandBye) {
				panic(fmt.Errorf("expected OK command from client, received %s", response.Command))
			} else if response.Command == protocol.CommandBye {
				return
			} else if err != nil {
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

	tcpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", port))

	ln, err := net.ListenTCP("tcp", tcpAddr)

	if err != nil {
		panic(err)
	}

	go func() {
		defer t.WaitGroup.Done()

		conn, err := ln.AcceptTCP()

		if err != nil {
			panic(err)
		}

		defer func() {
			protocol.WriteNewMessage(conn, protocol.CommandBye, "")
			conn.Close()
		}()

		_, err = protocol.WriteNewMessage(conn, protocol.CommandHello, "")

		if err != nil {
			panic(err)
		}

		receiveChan := make(chan *protocol.Message)
		errorChan := make(chan error)

		go protocol.ReadToChannel(conn, receiveChan, errorChan)

		for {
			select {
			case message :=  <-receiveChan:
				switch message.Command {
				case protocol.CommandTargetLog:
					log := broker.TargetLog{}

					err = json.Unmarshal([]byte(message.Data), &log)

					if err != nil {
						panic(err)
					}

					t.ReceivedLogs = append(t.ReceivedLogs, &log)

					_, err := protocol.WriteNewMessage(conn, protocol.CommandOk, "")

					if err != nil {
						panic(err)
					}
				}
			case err := <-errorChan:
				if err != nil {
					conn.Close()
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
