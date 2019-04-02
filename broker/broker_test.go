package broker

import (
	"encoding/json"
	"fmt"
	"github.com/phayes/freeport"
	"github.com/pm-connect/log-shipper/config"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/limiter"
	"github.com/pm-connect/log-shipper/message"
	"github.com/pm-connect/log-shipper/protocol"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestNewBroker(t *testing.T) {
	broker := NewBroker(1)

	assert.Len(t, broker.Sources, 0)
	assert.Len(t, broker.Targets, 0)
	assert.IsType(t, &sync.WaitGroup{}, broker.WorkerWaitGroup)
	assert.IsType(t, &sync.WaitGroup{}, broker.SourceWaitGroup)
	assert.IsType(t, &sync.WaitGroup{}, broker.TargetWaitGroup)
	assert.IsType(t, &sync.WaitGroup{}, broker.GeneralWaitGroup)
	assert.Equal(t, 1, broker.NumWorkers)
	assert.IsType(t, make(chan interface{}), broker.WorkerStop)
	assert.IsType(t, make(chan interface{}), broker.SourceStop)
	assert.IsType(t, make(chan interface{}), broker.TargetStop)
}

func TestBroker_AddSource(t *testing.T) {
	broker := NewBroker(1)

	source := &Source{}

	broker.AddSource("test", source)

	assert.Len(t, broker.Sources, 1)
	assert.Equal(t, source, broker.Sources["test"])
}

func TestBroker_AddTarget(t *testing.T) {
	broker := NewBroker(1)

	target := &Target{}

	broker.AddTarget("test", target)

	assert.Len(t, broker.Targets, 1)
	assert.Equal(t, target, broker.Targets["test"])
}

func TestBroker_Start(t *testing.T) {
	broker := NewBroker(1)

	primarySourceServer := TestSource{}
	primaryTargetServer := TestTarget{}
	secondaryTargetServer := TestTarget{}

	primarySourceDetails, _ := primarySourceServer.Start()
	primaryTargetDetails, _ := primaryTargetServer.Start()
	secondaryTargetDetails, _ := secondaryTargetServer.Start()

	primarySource := Source{
		ConnectionDetails: primarySourceDetails,
		Targets:           []string{"primary"},
	}

	primaryTargetRateLimiter := limiter.New("test", 10, time.Second, 1*time.Second, 10)

	primaryTarget := Target{
		ConnectionDetails: primaryTargetDetails,
		Config: config.Target{
			Provider: "test",
			RateLimit: []config.RateLimit{
				{
					Throughput: "10KB",
					Mode: config.RateLimitMode{
						Type:     "average",
						Period:   "1s",
						Duration: 10,
					},
					BreachBehaviour: config.BreachBehaviour{
						Action:     "fallback",
						Target:     "secondary",
						AlertLevel: "CRITICAL",
					},
				},
			},
		},
		RateLimitRules: []RateLimitRule{
			{
				RateLimiter: primaryTargetRateLimiter,
				BreachBehaviour: config.BreachBehaviour{
					Action:     "fallback",
					Target:     "secondary",
					AlertLevel: "CRITICAL",
				},
			},
		},
	}

	secondaryTarget := Target{
		ConnectionDetails: secondaryTargetDetails,
		Config: config.Target{
			Provider: "test",
		},
	}

	broker.AddSource("primary", &primarySource)
	broker.AddTarget("primary", &primaryTarget)
	broker.AddTarget("secondary", &secondaryTarget)

	go func() {
		for range time.Tick(100 * time.Millisecond) {
			fmt.Println("triggering stop")
			broker.Stop()
			break
		}
	}()

	err := broker.Start()

	assert.Nil(t, err)
	assert.NotZero(t, len(primaryTargetServer.ReceivedLogs))
	assert.NotZero(t, len(secondaryTargetServer.ReceivedLogs))
}

type TestSource struct {
	SentLogs int
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
