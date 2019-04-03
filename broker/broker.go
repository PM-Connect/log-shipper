package broker

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pm-connect/log-shipper/config"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/limiter"
	"github.com/pm-connect/log-shipper/message"
	"github.com/pm-connect/log-shipper/protocol"
)

type Broker struct {
	Sources    map[string]*Source
	Targets    map[string]*Target
	NumWorkers int

	// Channels to trigger the stop of workers/sources/targets.
	WorkerStop chan interface{}
	SourceStop chan interface{}
	TargetStop chan interface{}

	// WaitGroups to ensure graceful halting of processes.
	GeneralWaitGroup *sync.WaitGroup
	WorkerWaitGroup  *sync.WaitGroup
	SourceWaitGroup  *sync.WaitGroup
	TargetWaitGroup  *sync.WaitGroup
	FallbackTargetWaitGroup *sync.WaitGroup

	targetMessagesInFlight uint64
}

type Source struct {
	ConnectionDetails *connection.Details
	Targets           []string
}

type Target struct {
	ConnectionDetails *connection.Details
	Config            config.Target
	RateLimitRules    []RateLimitRule
}

type RateLimitRule struct {
	RateLimiter     *limiter.RateLimiter
	BreachBehaviour config.BreachBehaviour
}

type TargetChannels map[string]chan []byte

func NewBroker(workers int) *Broker {
	broker := Broker{}

	broker.Sources = map[string]*Source{}
	broker.Targets = map[string]*Target{}
	broker.WorkerStop = make(chan interface{})
	broker.SourceStop = make(chan interface{})
	broker.TargetStop = make(chan interface{})
	broker.NumWorkers = workers
	broker.GeneralWaitGroup = &sync.WaitGroup{}
	broker.WorkerWaitGroup = &sync.WaitGroup{}
	broker.SourceWaitGroup = &sync.WaitGroup{}
	broker.TargetWaitGroup = &sync.WaitGroup{}
	broker.FallbackTargetWaitGroup = &sync.WaitGroup{}

	return &broker
}

func (b *Broker) AddSource(name string, source *Source) {
	b.Sources[name] = source
}

func (b *Broker) AddTarget(name string, target *Target) {
	b.Targets[name] = target
}

func (b *Broker) Start() error {
	receiver := make(chan []byte)

	listeners := b.connectToTargets(receiver)

	b.connectToSources(receiver)
	b.workSources(receiver, listeners)

	b.SourceWaitGroup.Wait()
	close(b.WorkerStop)
	b.WorkerWaitGroup.Wait()
	for atomic.LoadUint64(&b.targetMessagesInFlight) > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	b.FallbackTargetWaitGroup.Wait()
	close(b.TargetStop)
	b.TargetWaitGroup.Wait()
	b.GeneralWaitGroup.Wait()

	return nil
}

func (b *Broker) Stop() {
	close(b.SourceStop)
}

func (b *Broker) connectToTargets(receiver chan []byte) *TargetChannels {
	targetChannels := TargetChannels{}

	for name, target := range b.Targets {
		b.TargetWaitGroup.Add(1)
		b.GeneralWaitGroup.Add(1)
		listen := make(chan []byte)
		go b.openTargetConnection(name, target, listen, receiver, &targetChannels)
		targetChannels[name] = listen
	}

	return &targetChannels
}

func (b *Broker) connectToSources(receiver chan []byte) {
	for name, source := range b.Sources {
		b.SourceWaitGroup.Add(1)
		go b.openSourceConnection(name, source, receiver)
	}
}

func (b *Broker) workSources(receiver chan []byte, targets *TargetChannels) {
	for i := 0; i < b.NumWorkers; i++ {
		b.WorkerWaitGroup.Add(1)
		go b.workReceiver(receiver, targets)
	}
}

func (b *Broker) workReceiver(receiver chan []byte, targets *TargetChannels) {
	defer b.WorkerWaitGroup.Done()

	for {
		select {
		case data := <-receiver:
			log, err := message.JsonToBroker(data)
			if err != nil {
				continue
			}

			targetChannels := *targets
			for _, target := range log.Targets {
				channel, ok := targetChannels[target]
				if ok {
					channel <- data
				}
			}
		case <-b.WorkerStop:
			return
		}
	}
}

func (b *Broker) openTargetConnection(name string, target *Target, listen <-chan []byte, receiver chan<- []byte, targets *TargetChannels) {
	defer b.GeneralWaitGroup.Done()
	defer b.TargetWaitGroup.Done()

	conn, err := connection.OpenTCPConnection(target.ConnectionDetails.Host, target.ConnectionDetails.Port)
	if err != nil {
		panic(err)
	}

	defer func() {
		protocol.SendBye(conn)
		_ = conn.Close()
	}()

	ready := protocol.SendHello(conn)

	if !ready {
		panic(fmt.Errorf("did not receive HELLO from server"))
	}

ReceiveLoop:
	for {
		select {
		case data := <-listen:
			atomic.AddUint64(&b.targetMessagesInFlight, 1)
			log, err := message.JsonToBroker(data)
			if err != nil {
				panic(err)
			}

			targetMessage := message.BrokerToTarget(name, log)

			rawData, err := json.Marshal(targetMessage)
			if err != nil {
				panic(err)
			}

			if passed := b.handleRateLimiting(log, target, targets); !passed {
				atomic.AddUint64(&b.targetMessagesInFlight, ^uint64(0))
				continue ReceiveLoop
			}

			_, err = protocol.WriteNewMessage(conn, protocol.CommandTargetLog, string(rawData))
			if err != nil {
				panic(err)
			}

			ok := protocol.WaitForOk(conn)
			if !ok {
				atomic.AddUint64(&b.targetMessagesInFlight, ^uint64(0))
				return
			}

			for _, rule := range target.RateLimitRules {
				bytes := uint64(len(rawData))
				rule.RateLimiter.Increment(bytes)
			}
			atomic.AddUint64(&b.targetMessagesInFlight, ^uint64(0))
		case <-b.TargetStop:
			protocol.SendBye(conn)
			_ = conn.Close()
			return
		}
	}
}

func (b *Broker) handleRateLimiting(message *message.BrokerMessage, target *Target, targets *TargetChannels) bool {
	for _, rule := range target.RateLimitRules {
		if _, _, over := rule.RateLimiter.IsAverageOverLimit(); over {
			// TODO: Send Alerts

			switch rule.BreachBehaviour.Action {
			case "discard":
				return false
			case "fallback":
				b.FallbackTargetWaitGroup.Add(1)
				newTargets := []string{rule.BreachBehaviour.Target}
				message.Targets = newTargets

				jsonData, err := json.Marshal(message)
				if err != nil {
					panic(err)
				}

				targetChannels := *targets

				channel, ok := targetChannels[rule.BreachBehaviour.Target]
				if ok {
					channel <- jsonData
				}

				b.FallbackTargetWaitGroup.Done()

				return false
			}
			break
		}
	}

	return true
}

func (b *Broker) openSourceConnection(name string, source *Source, receiver chan []byte) {
	defer b.SourceWaitGroup.Done()
	conn, err := connection.OpenTCPConnection(source.ConnectionDetails.Host, source.ConnectionDetails.Port)
	if err != nil {
		panic(err)
	}

	ready := protocol.SendHello(conn)
	if !ready {
		panic(fmt.Errorf("failed to send hello to source"))
	}

	receiveChan := make(chan *protocol.Message)
	errorChan := make(chan error)

	go protocol.ReadToChannel(conn, receiveChan, errorChan)

	defer func() {
		protocol.SendBye(conn)
		_ = conn.Close()
	}()

	for {
		select {
		case msg := <-receiveChan:
			switch msg.Command {
			case protocol.CommandSourceMessage:
				sourceMsg, err := message.JsonToSource(msg.Data)
				if err != nil {
					panic(err)
				}

				log := message.SourceToBroker(name, source.Targets, sourceMsg)

				jsonData, err := json.Marshal(log)
				if err != nil {
					panic(err)
				}

				protocol.SendOk(conn)

				receiver <- jsonData
			}
		case err := <-errorChan:
			if err != nil && err == io.EOF {
				return
			} else if err != nil {
				panic(err)
			}
		case <-b.SourceStop:
			_ = conn.Close()
			return
		}
	}
}
