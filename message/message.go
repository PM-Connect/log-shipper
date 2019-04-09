//go:generate protoc --go_out=. protobuf.proto

package message

import (
	"github.com/golang/protobuf/proto"
)

func BrokerToTarget(target string, b *BrokerMessage) *TargetMessage {
	return &TargetMessage{
		SourceMessage: b.SourceMessage,
		Target:        target,
		Source:        b.Source,
	}
}

func SourceToBroker(source string, targets []string, s *SourceMessage) *BrokerMessage {
	return &BrokerMessage{
		SourceMessage: s,
		Source:        source,
		Targets:       targets,
	}
}

func ProtobufToBroker(data []byte) (*BrokerMessage, error) {
	msg := &BrokerMessage{}

	err := proto.Unmarshal(data, msg)

	return msg, err
}

func ProtobufToSource(data []byte) (*SourceMessage, error) {
	msg := &SourceMessage{}

	err := proto.Unmarshal(data, msg)

	return msg, err
}

func ProtobufToTarget(data []byte) (*TargetMessage, error) {
	msg := &TargetMessage{}

	err := proto.Unmarshal(data, msg)

	return msg, err
}

func ToProtobuf(msg proto.Message) ([]byte, error) {
	return proto.Marshal(msg)
}
