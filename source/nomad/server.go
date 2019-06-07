package nomad

import (
	"bufio"
	"github.com/phayes/freeport"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/message"
	"github.com/pm-connect/log-shipper/protocol"
	"net"
)

type Source struct {
	Config map[string]string
	ConsulAddr string
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
		for {
			conn, err := ln.AcceptTCP()
			if err != nil {
				panic(err)
			}

			go func(conn net.Conn) {
				defer conn.Close()

				reader := bufio.NewReader(conn)

				ready := protocol.WaitForHello(reader)
				if !ready {
					// Received non-hello message.
					return
				}

				nomadClient, err := NewClient(s.Config, s.ConsulAddr)
				if err != nil {
					panic(err)
				}

				receiver := make(chan *message.SourceMessage)

				defer close(receiver)

				go nomadClient.ReceiveLogs(receiver)

				for log := range receiver {
					data, err := message.ToProtobuf(log)
					if err != nil {
						panic(err)
					}

					msg := &protocol.Message{
						Command: protocol.CommandSourceMessage,
						Data: []byte(string(data)),
					}

					bytes, err := protocol.MessageToProtobuf(msg)
					if err != nil {
						panic(err)
					}

					err = protocol.WriteProtobuf(conn, bytes)
					if err != nil {
						panic(err)
					}

					ok := protocol.WaitForOk(reader)
					if !ok {
						break
					}
				}
			}(conn)
		}
	}()

	return &connection.Details{
		Host: "127.0.0.1",
		Port: port,
	}, nil
}
