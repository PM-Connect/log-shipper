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
	GeneralWaitGroup   *sync.WaitGroup
	WorkerWaitGroup    *sync.WaitGroup
	SourceWaitGroup    *sync.WaitGroup
	TargetWaitGroup    *sync.WaitGroup
	ProcessedByWorker int
	ReceivedFromSources int
	SentToTargets int
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
	broker.GeneralWaitGroup = &sync.WaitGroup{}
	broker.WorkerWaitGroup = &sync.WaitGroup{}
	broker.SourceWaitGroup = &sync.WaitGroup{}
	broker.TargetWaitGroup = &sync.WaitGroup{}

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

	b.GeneralWaitGroup.Wait()

	return nil
}

func (b *Broker) Stop() {
	close(b.SourceStop)
	b.SourceWaitGroup.Wait()
	close(b.WorkerStop)
	b.WorkerWaitGroup.Wait()
	close(b.TargetStop)
	b.TargetWaitGroup.Wait()
}

func (b *Broker) connectToTargets(receiver chan []byte) *TargetChannels {
	targetChannels := TargetChannels{}

	for name, target := range b.Targets {
		b.TargetWaitGroup.Add(1)
		b.GeneralWaitGroup.Add(1)
		listen := make(chan []byte)
		go b.openTargetConnection(name, target, listen, receiver)
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
			b.ProcessedByWorker++

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
	defer b.GeneralWaitGroup.Done()
	defer b.TargetWaitGroup.Done()
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

	if message.Command != protocol.CommandHello {
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

			b.SentToTargets++

			response, err := protocol.ReadMessage(conn)

			if err == nil && (response.Command != protocol.CommandOk && response.Command != protocol.CommandBye) {
				panic(fmt.Errorf("expected OK command from client, received %s", response.Command))
			} else if response.Command == protocol.CommandBye {
				return
			} else if err != nil {
				panic(err)
			}
		case <-b.TargetStop:
			conn.Close()
			return
		}
	}
}

func (b *Broker) openSourceConnection(name string, source *Source, receiver chan []byte) {
	defer b.SourceWaitGroup.Done()
	tcpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", source.ConnectionDetails.Host, source.ConnectionDetails.Port))
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		panic(err)
	}

	_, err = protocol.WriteNewMessage(conn, protocol.CommandHello, "")

	if err != nil && err == io.EOF {
		return
	} else if err != nil {
		panic(err)
	}

	receiveChan := make(chan *protocol.Message)
	errorChan := make(chan error)

	go protocol.ReadToChannel(conn, receiveChan, errorChan)

	defer func() {
		protocol.WriteNewMessage(conn, protocol.CommandBye, "")
		conn.Close()
	}()

	for {
		select {
		case message := <-receiveChan:
			switch message.Command {
			case protocol.CommandSourceLog:
				b.ReceivedFromSources++
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

				_, _ = protocol.WriteNewMessage(conn, protocol.CommandOk, "")

				receiver <- jsonData
			}
		case err := <-errorChan:
			if err != nil && err == io.EOF {
				return
			} else if err != nil {
				panic(err)
			}
		case <-b.SourceStop:
			conn.Close()
			return
		}
	}
}
