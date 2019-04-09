package blackhole

import (
	"bufio"
	"fmt"
	"github.com/phayes/freeport"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/message"
	"github.com/pm-connect/log-shipper/protocol"
)

type Target struct {
}

func (t *Target) Start() (*connection.Details, error) {
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

		reader := bufio.NewReader(conn)

		ready := protocol.WaitForHello(reader)

		if !ready {
			panic(fmt.Errorf("failed to receive HELLO from broker"))
		}

		receiveChan := make(chan *protocol.Message)
		errorChan := make(chan error)

		go protocol.ReadToChannel(reader, receiveChan, errorChan)

		for {
			select {
			case msg := <-receiveChan:
				switch msg.Command {
				case protocol.CommandTargetLog:
					_, err := message.ProtobufToTarget(msg.Data)

					if err != nil {
						continue
					}

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