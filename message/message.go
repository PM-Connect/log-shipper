package message

import "encoding/json"

type BrokerMessage struct {
	Source        string
	SourceMessage *SourceMessage
	Targets       []string
}

type SourceMessage struct {
	ID      string
	Message string
	Meta    map[string]string
}

type TargetMessage struct {
	Target        string
	SourceMessage *SourceMessage
	Source        string
}

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

func JsonToBroker(data []byte) (*BrokerMessage, error) {
	msg := &BrokerMessage{}
	err := json.Unmarshal(data, msg)
	return msg, err
}

func JsonToSource(data []byte) (*SourceMessage, error) {
	msg := &SourceMessage{}
	err := json.Unmarshal(data, msg)
	return msg, err
}
