package nomad

import (
	"bufio"
	"fmt"
	"github.com/phayes/freeport"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/message"
	"github.com/pm-connect/log-shipper/protocol"
)

type Source struct {
	Config map[string]string
}

func (s *Source) Start() (*connection.Details, error) {
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

		reader := bufio.NewReader(conn)

		ready := protocol.WaitForHello(reader)

		if !ready {
			panic(fmt.Errorf("failed to receive HELLO from broker"))
		}

		nomadClient, err := NewClient(s.Config)

		if err != nil {
			panic(err)
		}

		receiver := make(chan []byte)

		defer close(receiver)

		go nomadClient.ReceiveLogs(receiver)

		for data := range receiver {
			log := &message.SourceMessage{
				Message: data,
			}

			data, err := message.ToProtobuf(log)

			if err != nil {
				continue
			}

			_, err = protocol.WriteNewMessage(conn, protocol.CommandSourceMessage, string(data))

			if err != nil {
				continue
			}

			ok := protocol.WaitForOk(reader)

			if !ok {
				return
			}
		}
	}()

	return &connection.Details{
		Host: "127.0.0.1",
		Port: port,
	}, nil
}
