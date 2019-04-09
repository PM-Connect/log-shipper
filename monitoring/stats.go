package monitoring

import "sync/atomic"

type Stats struct {
	BytesProcessed   uint64
	MessagesInbound  uint64
	MessagesOutbound uint64
	MessagesInFlight uint64
	DroppedMessages  uint64
	ResentMessages   uint64
}

func (s *Stats) IncrementBytes(delta uint64) {
	atomic.AddUint64(&s.BytesProcessed, delta)
}

func (s *Stats) IncrementMessagesInbound(delta uint64) {
	atomic.AddUint64(&s.MessagesInbound, delta)
}

func (s *Stats) IncrementMessagesOutbound(delta uint64) {
	atomic.AddUint64(&s.MessagesOutbound, delta)
}

func (s *Stats) IncrementDroppedMessages(delta uint64) {
	atomic.AddUint64(&s.DroppedMessages, delta)
}

func (s *Stats) IncrementResentMessages(delta uint64) {
	atomic.AddUint64(&s.ResentMessages, delta)
}

func (s *Stats) AddInFlightMessage() {
	atomic.AddUint64(&s.MessagesInFlight, uint64(1))
}

func (s *Stats) RemoveInFlightMessage() {
	atomic.AddUint64(&s.MessagesInFlight, ^uint64(0))
}

func (s *Stats) GetBytesProcessed() uint64 {
	return atomic.LoadUint64(&s.BytesProcessed)
}

func (s *Stats) GetMessagesInbound() uint64 {
	return atomic.LoadUint64(&s.MessagesInbound)
}

func (s *Stats) GetMessagesOutbound() uint64 {
	return atomic.LoadUint64(&s.MessagesOutbound)
}

func (s *Stats) GetInFlightMessages() uint64 {
	return atomic.LoadUint64(&s.MessagesInFlight)
}

func (s *Stats) GetDroppedMessages() uint64 {
	return atomic.LoadUint64(&s.DroppedMessages)
}

func (s *Stats) GetResentMessages() uint64 {
	return atomic.LoadUint64(&s.ResentMessages)
}
