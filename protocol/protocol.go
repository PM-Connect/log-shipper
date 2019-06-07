//go:generate protoc --go_out=. protobuf.proto

package protocol

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"io"
)

const CommandHello = "HELLO"
const CommandOk = "OK"
const CommandSourceMessage = "SMSG"
const CommandTargetLog = "TMSG"
const CommandBye = "BYE"

// ReadMessage reads from an instance of io.Reader and returns a single message when found.
func ReadMessage(reader *bufio.Reader) (*Message, error) {
	message := &Message{}

	sizeBytes := make([]byte, 4)

	_, err := reader.Read(sizeBytes)
	if err != nil {
		return message, err
	}

	size := binary.LittleEndian.Uint32(sizeBytes)

	data := make([]byte, size)

	_, err = reader.Read(data)
	if err != nil {
		return message, err
	}

	err = proto.Unmarshal(data, message)

	return message, err
}

func MessageToProtobuf(message *Message) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func WriteProtobuf(conn io.Writer, data []byte) error {
	size := uint32(len(data))

	b := []byte("")

	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.LittleEndian, size)
	if err != nil {
		return err
	}

	b = append(b[:], buff.Bytes()[:]...)
	b = append(b[:], data[:]...)

	_, err = conn.Write(b)

	return err
}

// WriteMessage sends a message to an instance of io.Writer.
func WriteMessage(conn io.Writer, message *Message) error {
	data, err := MessageToProtobuf(message)
	if err != nil {
		return err
	}

	return WriteProtobuf(conn, data)
}

// WriteNewMessage creates a message and sends it to the instance of io.Writer, then returns the created message.
func WriteNewMessage(conn io.Writer, command string, data string) (*Message, error) {
	message := &Message{
		Command: command,
		Data:    []byte(data),
	}

	return message, WriteMessage(conn, message)
}

// ReadToChannel reads an instance of io.Reader and sends all messages received to a given channel.
// Any errors received are sent to the errorChan.
func ReadToChannel(reader *bufio.Reader, messageChan chan<- *Message, errorChan chan<- error) {
	for {
		message, err := ReadMessage(reader)

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

func WaitForHello(reader *bufio.Reader) bool {
	message, err := ReadMessage(reader)

	if err != nil || message.Command != CommandHello {
		return false
	}

	return true
}

func SendHello(conn io.Writer) bool {
	_, err := WriteNewMessage(conn, CommandHello, "")

	if err != nil {
		return false
	}

	return true
}

func WaitForOk(reader *bufio.Reader) bool {
	message, err := ReadMessage(reader)

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
