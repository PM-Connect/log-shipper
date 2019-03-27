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
const CommandSourceMessage = "SMSG"
const CommandTargetLog = "TMSG"
const CommandBye = "BYE"

// Message is the data of a full message for the protocol.
type Message struct {
	Command string
	Data []byte
}

// ParseBytes parses a set of bytes into a message.
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
		Data: decoded,
	}, nil
}

// ReadMessage reads from an instance of io.Reader and returns a single message when found.
func ReadMessage(conn io.Reader) (*Message, error) {
	data, err := bufio.NewReader(conn).ReadBytes('\n')

	if err != nil {
		return &Message{}, err
	}

	message, err := ParseBytes(data)

	return &message, nil
}

// WriteMessage sends a message to an instance of io.Writer.
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

// WriteNewMessage creates a message and sends it to the instance of io.Writer, then returns the created message.
func WriteNewMessage(conn io.Writer, command string, data string) (*Message, error) {
	message := Message{
		Command: command,
		Data: []byte(data),
	}

	return &message, WriteMessage(conn, &message)
}

// ReadToChannel reads an instance of io.Reader and sends all messages received to a given channel.
// Any errors received are sent to the errorChan.
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

// WriteFromChannel listens to a channel and sends any messages received to the instance of io.Writer.
func WriteFromChannel(conn io.Writer, messageChan <-chan *Message, errorChan chan<- error) {
	for {
		message := <-messageChan

		err := WriteMessage(conn, message)

		if err != nil {
			errorChan <- err
		}
	}
}

func WaitForHello(conn io.Reader) bool {
	message, err := ReadMessage(conn)

	if err != nil || message.Command != CommandHello {
		return false
	}

	return true
}

func SendHello(conn io.Writer) bool {
	_, err := WriteNewMessage(conn, CommandHello, "")

	if err != nil  {
		return false
	}

	return true
}

func WaitForOk(conn io.Reader) bool {
	message, err := ReadMessage(conn)

	if err != nil || message.Command != CommandOk {
		return false
	}

	return true
}

func SendOk(conn io.Writer) {
	_, _ = WriteNewMessage(conn, CommandOk, "")
}

func SendBye(conn io.Writer) {
	_, _ = WriteNewMessage(conn, CommandBye, "")
}
