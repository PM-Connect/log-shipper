package broker

import (
	"encoding/json"
	"fmt"
	"github.com/pm-connect/log-shipper/config"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/protocol"
	"io"
	"net"
	"sync"
)

type Broker struct {
	Sources            map[string]*Source
	Targets            map[string]*Target
	NumWorkers         int
	WorkerStop         chan interface{}
	SourceStop         chan interface{}
	TargetStop         chan interface{}
	WaitGroup          *sync.WaitGroup
}

type Source struct {
	ConnectionDetails *connection.Details
	Targets           []string
}

type Target struct {
	ConnectionDetails *connection.Details
	Config            config.Target
}

type Log struct {
	Source    string
	SourceLog SourceLog
	Targets   []string
}

type SourceLog struct {
	ID      string
	Message string
	Meta    map[string]string
}

type TargetLog struct {
	Target    string
	SourceLog SourceLog
	Source    string
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
	broker.WaitGroup = &sync.WaitGroup{}

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

	b.WaitGroup.Wait()

	return nil
}

func (b *Broker) Stop() {
	close(b.SourceStop)
	close(b.WorkerStop)
	close(b.TargetStop)
}

func (b *Broker) connectToTargets(receiver chan []byte) *TargetChannels {
	targetChannels := TargetChannels{}

	for name, target := range b.Targets {
		b.WaitGroup.Add(1)
		listen := make(chan []byte)
		go b.openTargetConnection(name, target, listen, receiver)
		targetChannels[name] = listen
	}

	return &targetChannels
}

func (b *Broker) connectToSources(receiver chan []byte) {
	for name, source := range b.Sources {
		go b.openSourceConnection(name, source, receiver)
	}
}

func (b *Broker) workSources(receiver chan []byte, targets *TargetChannels) {
	for i := 0; i < b.NumWorkers; i++ {
		go b.workReceiver(receiver, targets)
	}
}

func (b *Broker) workReceiver(receiver chan []byte, targets *TargetChannels) {
	for {
		select {
		case data := <-receiver:
			log := Log{}

			err := json.Unmarshal(data, &log)

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

func (b *Broker) openTargetConnection(name string, target *Target, listen <-chan []byte, receiver chan<- []byte) {
	defer b.WaitGroup.Done()
	tcpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", target.ConnectionDetails.Host, target.ConnectionDetails.Port))
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	message, err := protocol.ReadMessage(conn)

	if err != nil && err == io.EOF {
		return
	} else if err != nil {
		panic(err)
	}

	if message.Command != protocol.CommandOk {
		panic(fmt.Errorf("expected OK command from client, received %s", message.Command))
	}

	for {
		select {
		case data := <-listen:
			log := Log{}

			err := json.Unmarshal(data, &log)

			if err != nil {
				panic(err)
			}

			// Check Rate Limiting Here

			targetLog := TargetLog{
				SourceLog: log.SourceLog,
				Target:    name,
				Source:    log.Source,
			}

			rawData, err := json.Marshal(targetLog)

			if err != nil {
				panic(err)
			}

			_, err = protocol.WriteNewMessage(conn, protocol.CommandTargetLog, string(rawData))

			if err != nil {
				panic(err)
			}

			response, err := protocol.ReadMessage(conn)

			if err != nil && err == io.EOF {
				return
			} else if err != nil {
				panic(err)
			}

			if response.Command != protocol.CommandOk {
				panic(fmt.Errorf("expected OK command from client, received %s", response.Command))
			}
		case <-b.TargetStop:
			return
		}
	}
}

func (b *Broker) openSourceConnection(name string, source *Source, receiver chan []byte) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", source.ConnectionDetails.Host, source.ConnectionDetails.Port))
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	okMsg, err := protocol.WriteNewMessage(conn, protocol.CommandOk, "")

	if err != nil && err == io.EOF {
		return
	} else if err != nil {
		panic(err)
	}

	receiveChan := make(chan *protocol.Message)
	errorChan := make(chan error)

	go protocol.ReadToChannel(conn, receiveChan, errorChan)

	for {
		select {
		case message := <-receiveChan:
			switch message.Command {
			case protocol.CommandSourceLog:
				sourceLog := SourceLog{}

				err = json.Unmarshal([]byte(message.Data), &sourceLog)

				if err != nil {
					panic(err)
				}

				log := Log{
					SourceLog: sourceLog,
					Source: name,
					Targets: source.Targets,
				}

				jsonData, err := json.Marshal(log)

				if err != nil {
					panic(err)
				}

				receiver <- jsonData

				err = protocol.WriteMessage(conn, okMsg)

				if err != nil && err == io.EOF {
					return
				} else if err != nil {
					panic(err)
				}
			}
		case err := <-errorChan:
			if err != nil && err == io.EOF {
				return
			} else if err != nil {
				panic(err)
			}
		case <-b.SourceStop:
			return
		}
	}
}
