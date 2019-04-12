package dummy

import (
	"bufio"
	"fmt"
	"github.com/phayes/freeport"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/message"
	"github.com/pm-connect/log-shipper/protocol"
	"time"
)

type Source struct {
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

		for {
			log := &message.SourceMessage{
				Id:      "1",
				Message: []byte("Test"),
				Meta: map[string]string{
					"some-key": "some-value",
				},
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

			time.Sleep(10 * time.Millisecond)
		}
	}()

	return &connection.Details{
		Host: "127.0.0.1",
		Port: port,
	}, nil
}
