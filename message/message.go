package message

type BrokerMessage struct {
	Source    string
	SourceMessage SourceMessage
	Targets   []string
}

type SourceMessage struct {
	ID      string
	Message string
	Meta    map[string]string
}

type TargetMessage struct {
	Target    string
	SourceMessage SourceMessage
	Source    string
}