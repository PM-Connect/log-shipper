package protocol

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const CommandHello = "HELLO"
const CommandOk = "OK"
const CommandSourceLog = "SLOG"
const CommandTargetLog = "TLOG"
const CommandBye = "BYE"

type Message struct {
	Command string
	Data string
}

func ParseBytes(bytes []byte) (Message, error) {
	messageParts := strings.SplitN(strings.TrimSpace(string(bytes)), " ", 2)

	if len(messageParts) == 1 {
		return Message{
			Command: messageParts[0],
		}, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(messageParts[1])

	if err != nil {
		return Message{}, err
	}

	return Message{
		Command: messageParts[0],
		Data: string(decoded),
	}, nil
}

func ReadMessage(conn io.Reader) (*Message, error) {
	data, err := bufio.NewReader(conn).ReadBytes('\n')

	if err != nil {
		return &Message{}, err
	}

	message, err := ParseBytes(data)

	return &message, nil
}

func WriteMessage(conn io.Writer, message *Message) error {
	var data []byte

	if len(message.Data) == 0 {
		data = []byte(fmt.Sprintf("%s\n", message.Command))
	} else {
		data = []byte(fmt.Sprintf("%s %s\n", message.Command, base64.StdEncoding.EncodeToString([]byte(message.Data))))
	}

	_, err := conn.Write(data)

	return err
}

func WriteNewMessage(conn io.Writer, command string, data string) (*Message, error) {
	message := Message{
		Command: command,
		Data: data,
	}

	return &message, WriteMessage(conn, &message)
}

func ReadToChannel(conn io.Reader, messageChan chan<- *Message, errorChan chan<- error) {
	for {
		message, err := ReadMessage(conn)

		if err != nil {
			if err == io.EOF {
				return
			}
			errorChan <- err
			continue
		}

		messageChan <- message
	}
}
