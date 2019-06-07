package logzio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/logzio/logzio-go"
	"github.com/phayes/freeport"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/message"
	"github.com/pm-connect/log-shipper/protocol"
)

type Target struct {
	Config map[string]string
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

		l, err := logzio.New(
			t.Config["token"],
			logzio.SetUrl(t.Config["endpoint"]),
			logzio.SetDrainDuration(time.Second*30),
			logzio.SetTempDirectory("/tmp/log-shipper/logzio"),
			logzio.SetDrainDiskThreshold(99),
			logzio.SetCheckDiskSpace(true),
			logzio.SetDebug(os.Stdout),
		)

		for {
			select {
			case msg := <-receiveChan:
				switch msg.Command {
				case protocol.CommandTargetLog:
					msg, err := message.ProtobufToTarget(msg.Data)

					if err != nil {
						continue
					}

					logzioMessage := map[string]interface{}{
						"message": string(msg.SourceMessage.Message),
						"type":    msg.SourceMessage.Attributes.Type,
					}

					if msg.SourceMessage.Attributes.Timestamp != int64(0) {
						logzioMessage["@timestamp"] = time.Unix(msg.SourceMessage.Attributes.Timestamp, 0).Format("2006-01-02T15:04:05.999-0700")
					}

					for key, value := range msg.SourceMessage.Meta {
						logzioMessage[key] = value
					}

					data, err := json.Marshal(logzioMessage)

					if err != nil {
						panic(err)
					}

					err = l.Send(data)

					if err != nil {
						panic(err)
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
