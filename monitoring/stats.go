package monitoring

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sync/atomic"
)

type Stats struct {
	BytesProcessed   uint64
	MessagesInbound  uint64
	MessagesOutbound uint64
	MessagesInFlight uint64
	DroppedMessages  uint64
	ResentMessages   uint64

	bytesProcessedCounter   prometheus.Counter
	messagesInboundCounter  prometheus.Counter
	messagesOutboundCounter prometheus.Counter
	droppedMessagesCounter  prometheus.Counter
	resentMessagesCounter   prometheus.Counter
	messagesInFlightGauge   prometheus.Gauge
}

func NewStats(metricPrefix string, labels prometheus.Labels) *Stats {
	return &Stats{
		bytesProcessedCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name:        fmt.Sprintf("logshipper_%sbytes_processed", metricPrefix),
			Help:        "The total number of bytes processed.",
			ConstLabels: labels,
		}),
		messagesInboundCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name:        fmt.Sprintf("logshipper_%smessages_inbound", metricPrefix),
			Help:        "The total number of messages received.",
			ConstLabels: labels,
		}),
		messagesOutboundCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name:        fmt.Sprintf("logshipper_%smessages_outbound", metricPrefix),
			Help:        "The total number of messages sent.",
			ConstLabels: labels,
		}),
		droppedMessagesCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name:        fmt.Sprintf("logshipper_%sdropped_messages", metricPrefix),
			Help:        "The total number of messages dropped.",
			ConstLabels: labels,
		}),
		resentMessagesCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name:        fmt.Sprintf("logshipper_%sresent_messages", metricPrefix),
			Help:        "The total number of messages resent/re-queued.",
			ConstLabels: labels,
		}),
		messagesInFlightGauge: promauto.NewGauge(prometheus.GaugeOpts{
			Name:        fmt.Sprintf("logshipper_%smessages_inflight", metricPrefix),
			Help:        "The total number of messages in flight/being processed.",
			ConstLabels: labels,
		}),
	}
}

func (s *Stats) IncrementBytes(delta uint64) {
	atomic.AddUint64(&s.BytesProcessed, delta)
	s.bytesProcessedCounter.Add(float64(delta))
}

func (s *Stats) IncrementMessagesInbound(delta uint64) {
	atomic.AddUint64(&s.MessagesInbound, delta)
	s.messagesInboundCounter.Add(float64(delta))
}

func (s *Stats) IncrementMessagesOutbound(delta uint64) {
	atomic.AddUint64(&s.MessagesOutbound, delta)
	s.messagesOutboundCounter.Add(float64(delta))
}

func (s *Stats) IncrementDroppedMessages(delta uint64) {
	atomic.AddUint64(&s.DroppedMessages, delta)
	s.droppedMessagesCounter.Add(float64(delta))
}

func (s *Stats) IncrementResentMessages(delta uint64) {
	atomic.AddUint64(&s.ResentMessages, delta)
	s.resentMessagesCounter.Add(float64(delta))
}

func (s *Stats) AddInFlightMessage() {
	atomic.AddUint64(&s.MessagesInFlight, uint64(1))
	s.messagesInFlightGauge.Inc()
}

func (s *Stats) RemoveInFlightMessage() {
	atomic.AddUint64(&s.MessagesInFlight, ^uint64(0))
	s.messagesInFlightGauge.Dec()
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
